package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MaxStagedUploadBytes is the practical ceiling for a file transferred through Wails' bound-method call path: its 64MB assembled-body cap leaves roughly this much room for raw bytes after base64 overhead.
const MaxStagedUploadBytes = 45 * 1024 * 1024

// DefaultStagedUploadTTL is how long a staged upload survives before Cleanup removes it.
const DefaultStagedUploadTTL = 24 * time.Hour

// MaxChunkBytes is the hard per-chunk size cap enforced by WriteChunk.
const MaxChunkBytes = 8 * 1024 * 1024

// MaxChunkedUploadBytes is the sanity ceiling on totalChunks * MaxChunkBytes for a single chunked upload.
const MaxChunkedUploadBytes = 2 * 1024 * 1024 * 1024

const chunkedUploadDirName = "chunked"
const chunkedUploadMetaFile = "meta.json"
const chunkFilePrefix = "chunk-"

// chunkedUploadMeta is the persisted record of a chunked upload session.
type chunkedUploadMeta struct {
	Filename    string    `json:"filename"`
	TotalChunks int       `json:"totalChunks"`
	CreatedAt   time.Time `json:"createdAt"`
}

// StagedUploadService writes uploaded bytes to a server-local file so existing path-based methods can consume them unchanged.
type StagedUploadService struct {
	baseDir string
	ttl     time.Duration
	mu      sync.Mutex
}

// NewStagedUploadService creates baseDir/staged-uploads under userDataPath; ttl <= 0 uses DefaultStagedUploadTTL.
func NewStagedUploadService(userDataPath string, ttl time.Duration) (*StagedUploadService, error) {
	baseDir := filepath.Join(userDataPath, "staged-uploads")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create staged-uploads directory: %w", err)
	}
	if ttl <= 0 {
		ttl = DefaultStagedUploadTTL
	}
	return &StagedUploadService{baseDir: baseDir, ttl: ttl}, nil
}

var unsafeFilenameChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// sanitizeFilename strips path separators and unsafe characters from a client-supplied name.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	if name == "" || name == "." || name == ".." {
		name = "upload"
	}
	return name
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Stage writes data under baseDir/clientID/<random-id>/filename and returns its path. clientID may be empty; it only namespaces uploads for debugging, never a security boundary.
func (s *StagedUploadService) Stage(clientID, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("no data to stage")
	}
	if len(data) > MaxStagedUploadBytes {
		return "", fmt.Errorf("file exceeds %d byte upload limit", MaxStagedUploadBytes)
	}

	if clientID == "" {
		clientID = "local"
	}
	clientID = sanitizeFilename(clientID)

	id, err := randomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate upload id: %w", err)
	}

	dir := filepath.Join(s.baseDir, clientID, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	path := filepath.Join(dir, sanitizeFilename(filename))
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write staged upload: %w", err)
	}
	return path, nil
}

// chunkFileName returns the zero-padded chunk filename for an index, so numeric sort order matches lexical sort order.
func chunkFileName(index int) string {
	return fmt.Sprintf("%s%06d", chunkFilePrefix, index)
}

// chunkedUploadDir returns the session directory for a chunked upload under the sanitized clientID.
func (s *StagedUploadService) chunkedUploadDir(clientID, uploadID string) string {
	if clientID == "" {
		clientID = "local"
	}
	clientID = sanitizeFilename(clientID)
	uploadID = sanitizeFilename(uploadID)
	return filepath.Join(s.baseDir, clientID, chunkedUploadDirName, uploadID)
}

func readChunkedUploadMeta(dir string) (*chunkedUploadMeta, error) {
	data, err := os.ReadFile(filepath.Join(dir, chunkedUploadMetaFile))
	if err != nil {
		return nil, err
	}
	var meta chunkedUploadMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// receivedChunkIndices lists the chunk indices already written in dir, sorted ascending.
func receivedChunkIndices(dir string) ([]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}
		return nil, err
	}
	indices := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), chunkFilePrefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(entry.Name(), chunkFilePrefix))
		if err != nil {
			continue
		}
		indices = append(indices, n)
	}
	sort.Ints(indices)
	return indices, nil
}

// StartChunkedUpload creates a new chunked upload session, or resumes an existing one for the same uploadID.
// It returns the indices of chunks already received, empty for a fresh session.
func (s *StagedUploadService) StartChunkedUpload(clientID, uploadID, filename string, totalChunks int) ([]int, error) {
	if uploadID == "" {
		return nil, fmt.Errorf("uploadID is required")
	}
	if totalChunks <= 0 {
		return nil, fmt.Errorf("totalChunks must be positive")
	}
	if int64(totalChunks)*int64(MaxChunkBytes) > MaxChunkedUploadBytes {
		return nil, fmt.Errorf("upload exceeds %d byte upload limit", MaxChunkedUploadBytes)
	}

	dir := s.chunkedUploadDir(clientID, uploadID)

	if existing, err := readChunkedUploadMeta(dir); err == nil {
		if existing.Filename != filename || existing.TotalChunks != totalChunks {
			return nil, fmt.Errorf("uploadID %q is already in use by a different upload", uploadID)
		}
		return receivedChunkIndices(dir)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read existing upload session: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunked upload directory: %w", err)
	}
	meta := chunkedUploadMeta{Filename: sanitizeFilename(filename), TotalChunks: totalChunks, CreatedAt: time.Now()}
	data, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to encode upload session: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, chunkedUploadMetaFile), data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write upload session: %w", err)
	}
	return []int{}, nil
}

// WriteChunk stores one chunk of an in-progress chunked upload, atomically.
func (s *StagedUploadService) WriteChunk(clientID, uploadID string, chunkIndex int, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("no data in chunk")
	}
	if len(data) > MaxChunkBytes {
		return fmt.Errorf("chunk exceeds %d byte chunk limit", MaxChunkBytes)
	}

	dir := s.chunkedUploadDir(clientID, uploadID)
	meta, err := readChunkedUploadMeta(dir)
	if err != nil {
		return fmt.Errorf("unknown upload session %q: %w", uploadID, err)
	}
	if chunkIndex < 0 || chunkIndex >= meta.TotalChunks {
		return fmt.Errorf("chunk index %d out of range [0,%d)", chunkIndex, meta.TotalChunks)
	}

	finalPath := filepath.Join(dir, chunkFileName(chunkIndex))
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write chunk: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("failed to finalize chunk: %w", err)
	}
	return nil
}

// ReceivedChunks reports which chunk indices are currently stored for an in-progress upload.
func (s *StagedUploadService) ReceivedChunks(clientID, uploadID string) ([]int, error) {
	dir := s.chunkedUploadDir(clientID, uploadID)
	if _, err := readChunkedUploadMeta(dir); err != nil {
		return nil, fmt.Errorf("unknown upload session %q: %w", uploadID, err)
	}
	return receivedChunkIndices(dir)
}

// CompleteChunkedUpload assembles all received chunks into the final staged file, in order, streaming
// one chunk at a time so the full file is never held in memory. It removes the chunk session on success.
func (s *StagedUploadService) CompleteChunkedUpload(clientID, uploadID string) (string, error) {
	dir := s.chunkedUploadDir(clientID, uploadID)
	meta, err := readChunkedUploadMeta(dir)
	if err != nil {
		return "", fmt.Errorf("unknown upload session %q: %w", uploadID, err)
	}

	received, err := receivedChunkIndices(dir)
	if err != nil {
		return "", err
	}
	if len(received) != meta.TotalChunks {
		return "", fmt.Errorf("upload incomplete: have %d of %d chunks", len(received), meta.TotalChunks)
	}

	sanitizedClientID := clientID
	if sanitizedClientID == "" {
		sanitizedClientID = "local"
	}
	sanitizedClientID = sanitizeFilename(sanitizedClientID)

	id, err := randomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate upload id: %w", err)
	}
	destDir := filepath.Join(s.baseDir, sanitizedClientID, id)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}
	destPath := filepath.Join(destDir, meta.Filename)

	dest, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create staged upload file: %w", err)
	}
	defer dest.Close()

	for i := 0; i < meta.TotalChunks; i++ {
		chunkPath := filepath.Join(dir, chunkFileName(i))
		chunk, err := os.Open(chunkPath)
		if err != nil {
			return "", fmt.Errorf("failed to open chunk %d: %w", i, err)
		}
		_, copyErr := io.Copy(dest, chunk)
		chunk.Close()
		if copyErr != nil {
			return "", fmt.Errorf("failed to assemble chunk %d: %w", i, copyErr)
		}
	}

	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("failed to clean up upload session: %w", err)
	}
	return destPath, nil
}

// AbortChunkedUpload discards an in-progress chunked upload session.
func (s *StagedUploadService) AbortChunkedUpload(clientID, uploadID string) error {
	dir := s.chunkedUploadDir(clientID, uploadID)
	return os.RemoveAll(dir)
}

// Remove deletes a staged upload's containing directory; refuses any path outside baseDir.
func (s *StagedUploadService) Remove(path string) error {
	absBase, err := filepath.Abs(s.baseDir)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to remove path outside staged-uploads directory: %s", path)
	}
	return os.RemoveAll(filepath.Dir(absPath))
}

// Cleanup removes staged upload directories whose contents are older than the configured TTL, returning how many were removed.
func (s *StagedUploadService) Cleanup() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientDirs, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	removed := 0
	cutoff := time.Now().Add(-s.ttl)
	for _, clientDir := range clientDirs {
		if !clientDir.IsDir() {
			continue
		}
		clientPath := filepath.Join(s.baseDir, clientDir.Name())
		uploadDirs, err := os.ReadDir(clientPath)
		if err != nil {
			continue
		}
		for _, uploadDir := range uploadDirs {
			uploadPath := filepath.Join(clientPath, uploadDir.Name())

			if uploadDir.Name() == chunkedUploadDirName {
				sessionDirs, err := os.ReadDir(uploadPath)
				if err != nil {
					continue
				}
				for _, sessionDir := range sessionDirs {
					sessionPath := filepath.Join(uploadPath, sessionDir.Name())
					info, err := sessionDir.Info()
					if err != nil {
						continue
					}
					if info.ModTime().Before(cutoff) {
						if err := os.RemoveAll(sessionPath); err == nil {
							removed++
						}
					}
				}
				continue
			}

			info, err := uploadDir.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				if err := os.RemoveAll(uploadPath); err == nil {
					removed++
				}
			}
		}
	}
	return removed, nil
}
