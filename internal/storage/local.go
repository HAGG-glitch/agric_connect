package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalStorage struct {
	uploadDir string
}

func NewLocalStorage(uploadDir string) (*LocalStorage, error) {
	abs, err := filepath.Abs(uploadDir)
	if err != nil {
		return nil, fmt.Errorf("resolving upload dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("creating upload dir: %w", err)
	}
	return &LocalStorage{uploadDir: abs}, nil
}

func (s *LocalStorage) Save(ctx context.Context, input SaveObjectInput) (StoredObject, error) {
	fullPath := filepath.Join(s.uploadDir, input.Path)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return StoredObject{}, fmt.Errorf("creating subdirectory: %w", err)
	}

	if !strings.HasPrefix(filepath.Clean(fullPath), s.uploadDir) {
		return StoredObject{}, fmt.Errorf("path traversal detected")
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return StoredObject{}, fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	size, err := io.Copy(f, input.Content)
	if err != nil {
		os.Remove(fullPath)
		return StoredObject{}, fmt.Errorf("writing file: %w", err)
	}

	return StoredObject{
		Path:        input.Path,
		ContentType: input.ContentType,
		SizeBytes:   size,
	}, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.uploadDir, path)

	if !strings.HasPrefix(filepath.Clean(fullPath), s.uploadDir) {
		return fmt.Errorf("path traversal detected")
	}

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file: %w", err)
	}
	return nil
}

func (s *LocalStorage) SignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return path, nil
}

func (s *LocalStorage) FullPath(path string) string {
	return filepath.Join(s.uploadDir, path)
}
