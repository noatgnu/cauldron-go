//go:build server

package main

import (
	"encoding/json"
	"io/fs"
	"net/http"
)

const sessionCookieName = "cauldron_session"

// newServerHandler adds the /auth/* routes in front of the SPA fallback. Static assets stay
// public so the Angular shell can load and show its own login screen; authMiddleware protects
// the actual API surface (/wails/runtime).
func newServerHandler(app *App, assetsFS fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", handleAuthLogin(app))
	mux.HandleFunc("/auth/status", handleAuthStatus(app))
	mux.HandleFunc("/auth/logout", handleAuthLogout())
	mux.Handle("/", newSPAHandler(assetsFS))
	return mux
}

// authMiddleware gates /wails/runtime (all bound-method calls) behind the session cookie.
// Static assets and the /auth/* routes themselves pass through unauthenticated by design --
// see newServerHandler's comment and the cauldron-server plan doc for why.
func authMiddleware(app *App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wails/runtime" {
				next.ServeHTTP(w, r)
				return
			}

			<-app.ready
			if app.serverAuthService == nil || !isAuthenticated(app, r) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isAuthenticated(app *App, r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return app.serverAuthService.Verify(cookie.Value)
}

func handleAuthLogin(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		<-app.ready
		w.Header().Set("Content-Type", "application/json")
		if app.serverAuthService == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "server not ready"})
			return
		}

		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		if !app.serverAuthService.Verify(body.Token) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    body.Token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
			MaxAge:   60 * 60 * 24 * 30,
		})
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

func handleAuthStatus(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		<-app.ready
		w.Header().Set("Content-Type", "application/json")
		authenticated := app.serverAuthService != nil && isAuthenticated(app, r)
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": authenticated})
	}
}

func handleAuthLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   -1,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}
