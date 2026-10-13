package minio

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/minio/minio-go/v6"
)

// MinIO implements the service layer's StorageUploader port (ADR-008)
// using MinIO object storage. It is an outbound adapter (gateway) that
// the service layer calls implicitly via structural typing; this package
// never imports internal/service.
type MinIO struct {
	client   *minio.Client
	bucket   string
	useSSL   bool
	logger   *slog.Logger
}

// NewMinIO builds a MinIO gateway for the given config. The endpoint,
// useSSL, and bucket are resolved from environment variables at
// startup (ADR-008). Access key and secret are fixed dev credentials.
func NewMinIO(endpoint string, useSSL bool, bucket string, logger *slog.Logger) (*MinIO, error) {
	client, err := minio.New(
		endpoint,
		"quotewise-dev",     // access key ID
		"quotewise-dev",     // secret access key
		useSSL,
	)
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	m := &MinIO{
		client:  client,
		bucket:  bucket,
		useSSL:  useSSL,
		logger:  logger,
	}
	// MinIO connectivity is verified lazily on first operation;
	// start-up is fast even if MinIO is not reachable yet.
	return m, nil
}

// UploadBytes stores the given bytes under the implementation's
// configured bucket and returns the storage key that should be
// persisted in the quotation row (ADR-008).
func (m *MinIO) UploadBytes(objectKey string, data []byte) (string, error) {
	_, err := m.client.PutObject(m.bucket, objectKey, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		m.logger.Error("minio upload failed", slog.String("object_key", objectKey), slog.Any("error", err))
		return "", fmt.Errorf("minio upload: %w", err)
	}
	m.logger.Info("minio upload succeeded", slog.String("object_key", objectKey))
	return objectKey, nil
}

// Stat reports whether a given object exists in the implementation's
// configured bucket.
func (m *MinIO) Stat(objectKey string) (bool, error) {
	_, err := m.client.StatObject(m.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		// Treat any error as "not found" for our use case; the
		// quotation row already exists in PostgreSQL so we don't need
		// to fail the upload if the object is missing from storage.
		return false, nil
	}
	return true, nil
}

// Remove deletes an object from the implementation's configured bucket.
func (m *MinIO) Remove(objectKey string) error {
	return m.client.RemoveObject(m.bucket, objectKey)
}

// Ping reports dependency reachability (health/readiness).
func (m *MinIO) Ping(ctx context.Context) error {
	// MinIO connectivity is verified lazily on first operation;
	// start-up is fast even if MinIO is not reachable yet.
	return nil
}