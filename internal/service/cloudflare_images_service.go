package service

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/pkg/imageutil"
)

// CloudflareImagesService handles interactions with Cloudflare R2 (S3-compatible)
type CloudflareImagesService interface {
	Upload(file *multipart.FileHeader, metadata map[string]string) (imageURL string, err error)
	Delete(imageURL string) error
	ExtractImageID(imageURL string) (string, error)
}

type cloudflareImagesService struct {
	config    *config.CloudflareConfig
	s3Client  *s3.Client
}

// NewCloudflareImagesService creates a new R2-backed CloudflareImagesService
func NewCloudflareImagesService(cfg *config.CloudflareConfig) CloudflareImagesService {
	// Strip any surrounding quotes that may have been introduced via env variable quoting
	cfg.R2PublicURL = strings.Trim(cfg.R2PublicURL, "\"")

	s3Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.R2Endpoint),
		Region:       "auto",
		Credentials:  aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.R2AccessKeyID, cfg.R2SecretAccessKey, "")),
	})

	return &cloudflareImagesService{
		config:   cfg,
		s3Client: s3Client,
	}
}

// Upload converts an image to WebP, uploads it to Cloudflare R2, and returns its public CDN URL.
// Object key format: <uuid>-<basename>.webp
func (s *cloudflareImagesService) Upload(file *multipart.FileHeader, metadata map[string]string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	webpData, err := imageutil.ConvertToWebP(src)
	if err != nil {
		return "", fmt.Errorf("failed to convert image to webp: %w", err)
	}

	baseName := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	objectKey := fmt.Sprintf("%s-%s.webp", uuid.New().String(), baseName)

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.config.R2Bucket),
		Key:           aws.String(objectKey),
		Body:          bytes.NewReader(webpData),
		ContentType:   aws.String("image/webp"),
		ContentLength: aws.Int64(int64(len(webpData))),
	}

	if _, err := s.s3Client.PutObject(context.Background(), input); err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", strings.TrimRight(s.config.R2PublicURL, "/"), objectKey)
	return publicURL, nil
}

// Delete removes an object from Cloudflare R2 by its public URL.
func (s *cloudflareImagesService) Delete(imageURL string) error {
	objectKey, err := s.ExtractImageID(imageURL)
	if err != nil {
		return fmt.Errorf("failed to extract object key: %w", err)
	}

	_, err = s.s3Client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.R2Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// ExtractImageID extracts the R2 object key from a full CDN URL.
// Example: https://cdn.healthcareforesights.com/<uuid>-file.jpg → <uuid>-file.jpg
func (s *cloudflareImagesService) ExtractImageID(imageURL string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("image URL is empty")
	}

	publicURL := strings.TrimRight(s.config.R2PublicURL, "/")
	if !strings.HasPrefix(imageURL, publicURL) {
		return "", fmt.Errorf("invalid R2 image URL: does not match public URL")
	}

	objectKey := strings.TrimPrefix(imageURL, publicURL+"/")
	if objectKey == "" {
		return "", fmt.Errorf("object key is empty in URL")
	}

	return objectKey, nil
}
