package service

import (
	"testing"

	"github.com/healthcare-market-research/backend/internal/config"
)

// NOTE: CloudflareImagesService now uploads to Cloudflare R2 via the AWS S3
// SDK (see cloudflare_images_service.go) rather than calling the legacy
// Cloudflare Images HTTP API. Upload/Delete exercise the real S3 client and
// are not covered here without an S3-client seam to mock; ExtractImageID is
// the pure, deterministic piece and is covered below.

func TestCloudflareImagesService_ExtractImageID(t *testing.T) {
	cfg := &config.CloudflareConfig{
		R2PublicURL: "https://cdn.example.com/images",
	}

	service := NewCloudflareImagesService(cfg)

	tests := []struct {
		name      string
		imageURL  string
		wantID    string
		wantError bool
	}{
		{
			name:      "Valid URL",
			imageURL:  "https://cdn.example.com/images/2cdc28f0-017a-49c4-9ed7-87056c83901-report.webp",
			wantID:    "2cdc28f0-017a-49c4-9ed7-87056c83901-report.webp",
			wantError: false,
		},
		{
			name:      "Empty URL",
			imageURL:  "",
			wantID:    "",
			wantError: true,
		},
		{
			name:      "URL with different host",
			imageURL:  "https://example.com/image.jpg",
			wantID:    "",
			wantError: true,
		},
		{
			// ExtractImageID trims the public URL prefix followed by "/"; a
			// URL identical to the public URL (no trailing slash, no key)
			// doesn't match that "prefix + /" form, so TrimPrefix is a no-op
			// and the whole URL is returned as the object key rather than
			// erroring. This pins down that (slightly surprising) current
			// behavior rather than the ideal one.
			name:      "URL equal to public URL with no object key",
			imageURL:  "https://cdn.example.com/images",
			wantID:    "https://cdn.example.com/images",
			wantError: false,
		},
		{
			name:      "URL equal to public URL plus trailing slash with no object key",
			imageURL:  "https://cdn.example.com/images/",
			wantID:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := service.ExtractImageID(tt.imageURL)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractImageID() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("ExtractImageID() gotID = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestCloudflareImagesService_ExtractImageID_TrailingSlashOnPublicURL(t *testing.T) {
	cfg := &config.CloudflareConfig{
		R2PublicURL: "https://cdn.example.com/images/",
	}

	service := NewCloudflareImagesService(cfg)

	gotID, err := service.ExtractImageID("https://cdn.example.com/images/key-123.webp")
	if err != nil {
		t.Fatalf("ExtractImageID() unexpected error: %v", err)
	}
	if gotID != "key-123.webp" {
		t.Errorf("ExtractImageID() gotID = %v, want %v", gotID, "key-123.webp")
	}
}
