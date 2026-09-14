//go:build server

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/services"
)

func newTestAppWithAuth(t *testing.T) *App {
	t.Helper()
	db, err := services.NewDatabaseServiceV3(t.TempDir())
	if err != nil {
		t.Fatalf("NewDatabaseServiceV3: %v", err)
	}
	authService, err := services.NewServerAuthService(db)
	if err != nil {
		t.Fatalf("NewServerAuthService: %v", err)
	}
	app := &App{ready: make(chan bool), serverAuthService: authService, db: db}
	close(app.ready)
	return app
}

func newNotReadyApp() *App {
	app := &App{ready: make(chan bool)}
	close(app.ready)
	return app
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode JSON response %q: %v", rec.Body.String(), err)
	}
}

func TestHandleAuthLogin_CorrectTokenSetsCookie(t *testing.T) {
	app := newTestAppWithAuth(t)
	token := app.serverAuthService.Token()

	body, _ := json.Marshal(map[string]string{"token": token})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleAuthLogin(app)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].Value != token {
		t.Fatalf("expected a session cookie carrying the token, got %+v", cookies)
	}
	if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Errorf("expected HttpOnly + SameSite=Strict cookie, got %+v", cookies[0])
	}
}

func TestHandleAuthLogin_WrongTokenRejected(t *testing.T) {
	app := newTestAppWithAuth(t)

	body, _ := json.Marshal(map[string]string{"token": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleAuthLogin(app)(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("expected no cookie to be set for a wrong token")
	}
}

func TestHandleAuthLogin_RejectsNonPost(t *testing.T) {
	app := newTestAppWithAuth(t)
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()

	handleAuthLogin(app)(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandleAuthLogin_RejectsMalformedBody(t *testing.T) {
	app := newTestAppWithAuth(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()

	handleAuthLogin(app)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAuthStatus(t *testing.T) {
	app := newTestAppWithAuth(t)
	token := app.serverAuthService.Token()

	t.Run("no cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/status", nil)
		rec := httptest.NewRecorder()
		handleAuthStatus(app)(rec, req)

		var result map[string]bool
		decodeJSON(t, rec, &result)
		if result["authenticated"] {
			t.Error("expected authenticated=false with no cookie")
		}
	})

	t.Run("valid cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/status", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		rec := httptest.NewRecorder()
		handleAuthStatus(app)(rec, req)

		var result map[string]bool
		decodeJSON(t, rec, &result)
		if !result["authenticated"] {
			t.Error("expected authenticated=true with a valid cookie")
		}
	})

	t.Run("invalid cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/status", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "garbage"})
		rec := httptest.NewRecorder()
		handleAuthStatus(app)(rec, req)

		var result map[string]bool
		decodeJSON(t, rec, &result)
		if result["authenticated"] {
			t.Error("expected authenticated=false with an invalid cookie")
		}
	})
}

func TestHandleAuthLogout_ClearsCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	handleAuthLogout()(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected a clearing cookie (negative MaxAge), got %+v", cookies)
	}
}

func TestAuthMiddleware_ProtectsWailsRuntime(t *testing.T) {
	app := newTestAppWithAuth(t)
	token := app.serverAuthService.Token()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	handler := authMiddleware(app)(next)

	t.Run("no cookie is rejected", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodPost, "/wails/runtime", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
		if nextCalled {
			t.Error("expected next handler not to be called without a valid session")
		}
	})

	t.Run("valid cookie is allowed through", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodPost, "/wails/runtime", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
		if !nextCalled {
			t.Error("expected next handler to be called with a valid session")
		}
	})
}

func TestAuthMiddleware_StaticAssetsStayPublic(t *testing.T) {
	app := newTestAppWithAuth(t)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	handler := authMiddleware(app)(next)

	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !nextCalled {
		t.Errorf("expected static assets to pass through unauthenticated, status = %d, nextCalled = %v", rec.Code, nextCalled)
	}
}

func TestAuthMiddleware_NotReadyRejectsProtectedPath(t *testing.T) {
	app := newNotReadyApp()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := authMiddleware(app)(next)

	req := httptest.NewRequest(http.MethodPost, "/wails/runtime", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 when server auth isn't initialized yet", rec.Code)
	}
}
