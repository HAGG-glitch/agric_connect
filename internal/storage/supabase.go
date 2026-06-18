package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// NormalizePath strips accidental bucket prefix from a stored object path.
// The bucket is configured separately, so paths should not include it.
func NormalizePath(path, bucket string) string {
	path = strings.TrimLeft(path, "/")
	if bucket != "" {
		if strings.HasPrefix(path, bucket+"/") {
			path = strings.TrimPrefix(path, bucket+"/")
		} else if path == bucket {
			path = ""
		}
	}
	return path
}

// redactToken removes the token query parameter from a signed URL for logging.
func redactToken(url string) string {
	if idx := strings.Index(url, "token="); idx >= 0 {
		return url[:idx+6] + "<redacted>"
	}
	if idx := strings.Index(url, "?"); idx >= 0 {
		return url[:idx]
	}
	return url
}

type SupabaseStorage struct {
	url        string
	secretKey  string
	bucket     string
	httpClient *http.Client
}

func NewSupabaseStorage(url, secretKey, bucket string) *SupabaseStorage {
	return &SupabaseStorage{
		url:        url,
		secretKey:  secretKey,
		bucket:     bucket,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func NewSupabaseStorageWithClient(url, secretKey, bucket string, client *http.Client) *SupabaseStorage {
	return &SupabaseStorage{
		url:        url,
		secretKey:  secretKey,
		bucket:     bucket,
		httpClient: client,
	}
}

func (s *SupabaseStorage) Save(ctx context.Context, input SaveObjectInput) (StoredObject, error) {
	data, err := io.ReadAll(input.Content)
	if err != nil {
		return StoredObject{}, fmt.Errorf("reading content: %w", err)
	}

	if input.Path == "" {
		return StoredObject{}, fmt.Errorf("object path is required")
	}

	normalizedPath := NormalizePath(input.Path, s.bucket)
	log.Printf("storage_save: bucket=%s, original_path=%q, normalized_path=%q", s.bucket, input.Path, normalizedPath)

	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, normalizedPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return StoredObject{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("apikey", s.secretKey)
	if input.ContentType != "" {
		req.Header.Set("Content-Type", input.ContentType)
	} else {
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return StoredObject{}, fmt.Errorf("uploading to supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		log.Printf("storage_upload_failed: bucket=%s, path=%q, status=%d, body=%s", s.bucket, normalizedPath, resp.StatusCode, string(body))
		return StoredObject{}, fmt.Errorf("supabase upload failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	log.Printf("storage_upload_ok: bucket=%s, path=%q, size=%d", s.bucket, normalizedPath, int64(len(data)))

	return StoredObject{
		Path:        normalizedPath,
		ContentType: input.ContentType,
		SizeBytes:   int64(len(data)),
	}, nil
}

func (s *SupabaseStorage) Delete(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("object path is required")
	}

	normalizedPath := NormalizePath(path, s.bucket)
	delURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, normalizedPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, delURL, nil)
	if err != nil {
		return fmt.Errorf("creating delete request: %w", err)
	}
	req.Header.Set("apikey", s.secretKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting from supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("supabase delete failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *SupabaseStorage) SignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	if path == "" {
		return "", fmt.Errorf("object path is required")
	}

	normalizedPath := NormalizePath(path, s.bucket)
	log.Printf("storage_signed_url: bucket=%s, original_path=%q, normalized_path=%q", s.bucket, path, normalizedPath)

	signURL := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", s.url, s.bucket, normalizedPath)
	bodyPayload := map[string]int{"expiresIn": int(expiry.Seconds())}
	bodyBytes, _ := json.Marshal(bodyPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, signURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("creating signed url request: %w", err)
	}
	req.Header.Set("apikey", s.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("getting signed url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("supabase signed url failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		SignedURL string `json:"signedURL"`
		Symmetric string `json:"symmetric"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding signed url response: %w", err)
	}

	raw := result.SignedURL
	if raw == "" {
		raw = result.Symmetric
	}
	if raw == "" {
		return "", fmt.Errorf("supabase signed url response missing signedURL field")
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw, nil
	}

	baseURL := strings.TrimRight(s.url, "/")
	if strings.HasPrefix(raw, "/") {
		return baseURL + raw, nil
	}

	return baseURL + "/" + raw, nil
}

func (s *SupabaseStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, fmt.Errorf("object path is required")
	}

	normalizedPath := NormalizePath(path, s.bucket)
	downloadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, normalizedPath)
	log.Printf("storage_download: bucket=%s, normalized_path=%q, host=%s, path_without_token=%s",
		s.bucket, normalizedPath, s.url, redactToken(downloadURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("apikey", s.secretKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading from supabase: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("supabase download failed (HTTP 404): object not found at path %q", normalizedPath)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("supabase download failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}
