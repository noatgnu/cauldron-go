package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStagedUploadService(t *testing.T) *StagedUploadService {
	t.Helper()
	svc, err := NewStagedUploadService(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewStagedUploadService error: %v", err)
	}
	return svc
}

func TestStagedUploadService_StageWritesReadableFile(t *testing.T) {
	svc := newTestStagedUploadService(t)

	path, err := svc.Stage("client-1", "sample.tsv", []byte("a\tb\n1\t2\n"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read staged file: %v", err)
	}
	if string(data) != "a\tb\n1\t2\n" {
		t.Errorf("staged content = %q, want %q", data, "a\tb\n1\t2\n")
	}
	if filepath.Base(path) != "sample.tsv" {
		t.Errorf("staged filename = %q, want sample.tsv", filepath.Base(path))
	}
}

func TestStagedUploadService_StageRejectsEmptyData(t *testing.T) {
	svc := newTestStagedUploadService(t)
	if _, err := svc.Stage("client-1", "empty.txt", nil); err == nil {
		t.Error("expected an error staging empty data, got nil")
	}
}

func TestStagedUploadService_StageRejectsOversizedData(t *testing.T) {
	svc := newTestStagedUploadService(t)
	oversized := make([]byte, MaxStagedUploadBytes+1)
	if _, err := svc.Stage("client-1", "big.bin", oversized); err == nil {
		t.Error("expected an error staging data over the upload limit, got nil")
	}
}

func TestStagedUploadService_StageSanitizesTraversalFilename(t *testing.T) {
	svc := newTestStagedUploadService(t)

	path, err := svc.Stage("client-1", "../../../etc/passwd", []byte("data"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}

	if filepath.Base(path) != "passwd" {
		t.Errorf("expected traversal path to collapse to basename, got filename %q", filepath.Base(path))
	}
	if strings.Contains(path, "..") {
		t.Errorf("staged path still contains traversal segments: %s", path)
	}
}

func TestStagedUploadService_StageSanitizesTraversalClientID(t *testing.T) {
	svc := newTestStagedUploadService(t)

	path, err := svc.Stage("../../evil", "file.txt", []byte("data"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}
	if strings.Contains(path, "..") {
		t.Errorf("staged path still contains traversal segments from clientID: %s", path)
	}
}

func TestStagedUploadService_StageEmptyClientIDDefaultsToLocal(t *testing.T) {
	svc := newTestStagedUploadService(t)

	path, err := svc.Stage("", "file.txt", []byte("data"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}
	if !strings.Contains(filepath.ToSlash(path), "/local/") {
		t.Errorf("expected empty clientID to be staged under a 'local' directory, got path: %s", path)
	}
}

func TestStagedUploadService_RemoveDeletesStagedUpload(t *testing.T) {
	svc := newTestStagedUploadService(t)

	path, err := svc.Stage("client-1", "file.txt", []byte("data"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}

	if err := svc.Remove(path); err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected staged file to be gone after Remove, stat err = %v", err)
	}
}

func TestStagedUploadService_RemoveRefusesPathOutsideBaseDir(t *testing.T) {
	svc := newTestStagedUploadService(t)

	outside := filepath.Join(t.TempDir(), "not-a-staged-upload.txt")
	if err := os.WriteFile(outside, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	if err := svc.Remove(outside); err == nil {
		t.Error("expected Remove to refuse a path outside the staged-uploads directory, got nil error")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("expected the outside file to survive the refused Remove, stat err = %v", err)
	}
}

func TestStagedUploadService_CleanupRemovesOnlyExpiredUploads(t *testing.T) {
	svc := newTestStagedUploadService(t)

	oldPath, err := svc.Stage("client-1", "old.txt", []byte("data"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}
	freshPath, err := svc.Stage("client-1", "fresh.txt", []byte("data"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}

	oldDir := filepath.Dir(oldPath)
	backdated := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldDir, backdated, backdated); err != nil {
		t.Fatalf("failed to backdate fixture: %v", err)
	}

	removed, err := svc.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}
	if removed != 1 {
		t.Errorf("Cleanup removed = %d, want 1", removed)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("expected the expired upload to be removed, stat err = %v", err)
	}
	if _, err := os.Stat(freshPath); err != nil {
		t.Errorf("expected the fresh upload to survive Cleanup, stat err = %v", err)
	}
}

func TestStagedUploadService_CleanupOnMissingBaseDirIsNoop(t *testing.T) {
	svc := newTestStagedUploadService(t)
	if err := os.RemoveAll(svc.baseDir); err != nil {
		t.Fatalf("failed to remove baseDir fixture: %v", err)
	}

	removed, err := svc.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup error on missing baseDir: %v", err)
	}
	if removed != 0 {
		t.Errorf("Cleanup removed = %d, want 0", removed)
	}
}

func TestStagedUploadService_StageTwiceProducesDistinctPaths(t *testing.T) {
	svc := newTestStagedUploadService(t)

	path1, err := svc.Stage("client-1", "same-name.txt", []byte("data"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}
	path2, err := svc.Stage("client-1", "same-name.txt", []byte("data"))
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}

	if path1 == path2 {
		t.Error("expected two uploads with the same filename to be staged at distinct paths")
	}
}

func TestStagedUploadService_StartChunkedUploadFreshSessionReturnsEmpty(t *testing.T) {
	svc := newTestStagedUploadService(t)

	received, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 3)
	if err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if len(received) != 0 {
		t.Errorf("expected no received chunks for a fresh session, got %v", received)
	}
}

func TestStagedUploadService_StartChunkedUploadResumesExistingSession(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 3); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 0, []byte("chunk0")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 2, []byte("chunk2")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}

	received, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 3)
	if err != nil {
		t.Fatalf("StartChunkedUpload (resume) error: %v", err)
	}
	if len(received) != 2 || received[0] != 0 || received[1] != 2 {
		t.Errorf("expected resumed session to report [0 2], got %v", received)
	}
}

func TestStagedUploadService_StartChunkedUploadMismatchedSessionErrors(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 3); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "different.bin", 3); err == nil {
		t.Error("expected an error starting a session with the same uploadID but a different filename")
	}
	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 5); err == nil {
		t.Error("expected an error starting a session with the same uploadID but a different chunk count")
	}
}

func TestStagedUploadService_WriteChunkRejectsOutOfRangeIndex(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 2); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", -1, []byte("data")); err == nil {
		t.Error("expected an error writing a negative chunk index")
	}
	if err := svc.WriteChunk("client-1", "upload-1", 2, []byte("data")); err == nil {
		t.Error("expected an error writing a chunk index beyond totalChunks")
	}
}

func TestStagedUploadService_WriteChunkRejectsOversizedChunk(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 1); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	oversized := make([]byte, MaxChunkBytes+1)
	if err := svc.WriteChunk("client-1", "upload-1", 0, oversized); err == nil {
		t.Error("expected an error writing a chunk over the per-chunk limit")
	}
}

func TestStagedUploadService_WriteChunkUnknownSessionErrors(t *testing.T) {
	svc := newTestStagedUploadService(t)
	if err := svc.WriteChunk("client-1", "no-such-upload", 0, []byte("data")); err == nil {
		t.Error("expected an error writing a chunk for an unknown upload session")
	}
}

func TestStagedUploadService_WriteChunkIsIdempotent(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 1); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 0, []byte("first")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 0, []byte("second")); err != nil {
		t.Fatalf("WriteChunk (retry) error: %v", err)
	}

	path, err := svc.CompleteChunkedUpload("client-1", "upload-1")
	if err != nil {
		t.Fatalf("CompleteChunkedUpload error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read completed upload: %v", err)
	}
	if string(data) != "second" {
		t.Errorf("completed upload content = %q, want %q (last write wins)", data, "second")
	}
}

func TestStagedUploadService_ReceivedChunksReflectsWrites(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 4); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 3, []byte("d")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 1, []byte("b")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}

	received, err := svc.ReceivedChunks("client-1", "upload-1")
	if err != nil {
		t.Fatalf("ReceivedChunks error: %v", err)
	}
	if len(received) != 2 || received[0] != 1 || received[1] != 3 {
		t.Errorf("expected sorted [1 3], got %v", received)
	}
}

func TestStagedUploadService_CompleteChunkedUploadAssemblesInOrder(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "assembled.txt", 3); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 2, []byte("ghi")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 0, []byte("abc")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 1, []byte("def")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}

	path, err := svc.CompleteChunkedUpload("client-1", "upload-1")
	if err != nil {
		t.Fatalf("CompleteChunkedUpload error: %v", err)
	}
	if filepath.Base(path) != "assembled.txt" {
		t.Errorf("completed upload filename = %q, want assembled.txt", filepath.Base(path))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read completed upload: %v", err)
	}
	if string(data) != "abcdefghi" {
		t.Errorf("completed upload content = %q, want %q", data, "abcdefghi")
	}
	if _, err := os.Stat(svc.chunkedUploadDir("client-1", "upload-1")); !os.IsNotExist(err) {
		t.Errorf("expected chunked session directory to be removed after completion, stat err = %v", err)
	}
}

func TestStagedUploadService_CompleteChunkedUploadFailsIfIncomplete(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "assembled.txt", 2); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 0, []byte("abc")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}

	if _, err := svc.CompleteChunkedUpload("client-1", "upload-1"); err == nil {
		t.Error("expected CompleteChunkedUpload to fail with a missing chunk")
	}
}

func TestStagedUploadService_AbortChunkedUploadRemovesSession(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "upload-1", "big.bin", 2); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if err := svc.WriteChunk("client-1", "upload-1", 0, []byte("data")); err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}

	if err := svc.AbortChunkedUpload("client-1", "upload-1"); err != nil {
		t.Fatalf("AbortChunkedUpload error: %v", err)
	}
	if _, err := svc.ReceivedChunks("client-1", "upload-1"); err == nil {
		t.Error("expected the aborted session to be gone")
	}
}

func TestStagedUploadService_CleanupRemovesExpiredChunkedSessions(t *testing.T) {
	svc := newTestStagedUploadService(t)

	if _, err := svc.StartChunkedUpload("client-1", "old-upload", "big.bin", 2); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if _, err := svc.StartChunkedUpload("client-1", "fresh-upload", "big.bin", 2); err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}

	oldDir := svc.chunkedUploadDir("client-1", "old-upload")
	backdated := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldDir, backdated, backdated); err != nil {
		t.Fatalf("failed to backdate fixture: %v", err)
	}

	removed, err := svc.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}
	if removed != 1 {
		t.Errorf("Cleanup removed = %d, want 1", removed)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Errorf("expected the expired chunked session to be removed, stat err = %v", err)
	}
	if _, err := svc.ReceivedChunks("client-1", "fresh-upload"); err != nil {
		t.Errorf("expected the fresh chunked session to survive Cleanup: %v", err)
	}
}

func TestStagedUploadService_ChunkedUploadFullResumeFlow(t *testing.T) {
	svc := newTestStagedUploadService(t)

	chunks := [][]byte{[]byte("hello "), []byte("resumable "), []byte("world")}
	received, err := svc.StartChunkedUpload("client-1", "upload-1", "greeting.txt", len(chunks))
	if err != nil {
		t.Fatalf("StartChunkedUpload error: %v", err)
	}
	if len(received) != 0 {
		t.Fatalf("expected fresh session, got received chunks %v", received)
	}

	for i := 0; i < 2; i++ {
		if err := svc.WriteChunk("client-1", "upload-1", i, chunks[i]); err != nil {
			t.Fatalf("WriteChunk %d error: %v", i, err)
		}
	}

	resumed, err := svc.StartChunkedUpload("client-1", "upload-1", "greeting.txt", len(chunks))
	if err != nil {
		t.Fatalf("StartChunkedUpload (reconnect) error: %v", err)
	}
	if len(resumed) != 2 || resumed[0] != 0 || resumed[1] != 1 {
		t.Fatalf("expected resumed session to report [0 1], got %v", resumed)
	}

	if err := svc.WriteChunk("client-1", "upload-1", 2, chunks[2]); err != nil {
		t.Fatalf("WriteChunk 2 error: %v", err)
	}

	path, err := svc.CompleteChunkedUpload("client-1", "upload-1")
	if err != nil {
		t.Fatalf("CompleteChunkedUpload error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read completed upload: %v", err)
	}
	want := "hello resumable world"
	if string(data) != want {
		t.Errorf("completed upload content = %q, want %q", data, want)
	}
}
