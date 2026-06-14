package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

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

	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, input.Path)
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
		return StoredObject{}, fmt.Errorf("supabase upload failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return StoredObject{
		Path:        input.Path,
		ContentType: input.ContentType,
		SizeBytes:   int64(len(data)),
	}, nil
}

func (s *SupabaseStorage) Delete(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("object path is required")
	}

	delURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, path)
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

	signURL := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", s.url, s.bucket, path)
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

	if result.SignedURL != "" {
		return result.SignedURL, nil
	}
	if result.Symmetric != "" {
		return result.Symmetric, nil
	}

	return "", fmt.Errorf("supabase signed url response missing signedURL field")
}
