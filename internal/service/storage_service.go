package service

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

// StorageService handles file uploads to Cloudflare R2
type StorageService interface {
	UploadAvatar(ctx context.Context, userID uuid.UUID, fileData []byte, contentType string) (string, error)
	DeleteAvatar(ctx context.Context, avatarURL string) error
}

type storageService struct {
	s3Client   *s3.S3
	bucketName string
	publicURL  string
}

// NewStorageService creates a new storage service for R2
func NewStorageService(accountID, accessKeyID, secretAccessKey, bucketName, publicURL string) (StorageService, error) {
	// R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("auto"),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &storageService{
		s3Client:   s3.New(sess),
		bucketName: bucketName,
		publicURL:  publicURL,
	}, nil
}

// UploadAvatar uploads an avatar image to R2
func (s *storageService) UploadAvatar(ctx context.Context, userID uuid.UUID, fileData []byte, contentType string) (string, error) {
	// Generate unique filename
	ext := getExtensionFromContentType(contentType)
	filename := fmt.Sprintf("avatars/%s%s", uuid.New().String(), ext)

	// Upload to R2
	_, err := s.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"), // Make avatars publicly accessible
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Return public URL
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.publicURL, "/"), filename), nil
	}

	// Fallback to bucket URL if no custom domain
	return fmt.Sprintf("https://%s.r2.dev/%s", s.bucketName, filename), nil
}

// DeleteAvatar deletes an avatar from R2
func (s *storageService) DeleteAvatar(ctx context.Context, avatarURL string) error {
	if avatarURL == "" {
		return nil
	}

	// Extract filename from URL
	filename := extractFilenameFromURL(avatarURL)
	if filename == "" {
		return fmt.Errorf("invalid avatar URL")
	}

	_, err := s.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// getExtensionFromContentType maps MIME types to file extensions
func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

// extractFilenameFromURL extracts the object key from a full URL
func extractFilenameFromURL(url string) string {
	// Extract path after domain
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return ""
	}

	// Get last two parts: "avatars/uuid.jpg"
	if len(parts) >= 2 {
		return filepath.Join(parts[len(parts)-2], parts[len(parts)-1])
	}

	return parts[len(parts)-1]
}
