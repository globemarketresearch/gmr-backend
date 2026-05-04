package validation

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

const (
	// MaxImageSize is the maximum allowed image file size (10 MB)
	MaxImageSize = 10 * 1024 * 1024 // 10 MB in bytes
)

var (
	// AllowedImageTypes contains the allowed MIME types for images
	AllowedImageTypes = map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}

	// AllowedImageExtensions contains the allowed file extensions for images
	AllowedImageExtensions = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".gif":  true,
	}
)

// ValidateImageFile validates an uploaded image file
func ValidateImageFile(file *multipart.FileHeader) error {
	if file == nil {
		return fmt.Errorf("no file provided")
	}

	// Validate file size
	if file.Size > MaxImageSize {
		return fmt.Errorf("image size must be less than 10MB (current size: %.2f MB)", float64(file.Size)/(1024*1024))
	}

	if file.Size == 0 {
		return fmt.Errorf("image file is empty")
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !AllowedImageExtensions[ext] {
		return fmt.Errorf("invalid file extension: %s (allowed: .jpg, .jpeg, .png, .webp, .gif)", ext)
	}

	// Validate MIME type from header
	contentType := file.Header.Get("Content-Type")
	if contentType != "" && !AllowedImageTypes[contentType] {
		return fmt.Errorf("invalid file type: %s (allowed: image/jpeg, image/png, image/webp, image/gif)", contentType)
	}

	// Additional validation: Read file header to verify actual content type
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file for validation: %w", err)
	}
	defer src.Close()

	// Read the first 512 bytes to detect the content type
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// Detect MIME type from content
	detectedType := detectImageType(buffer[:n])
	if detectedType == "" {
		return fmt.Errorf("file does not appear to be a valid image")
	}

	if !AllowedImageTypes[detectedType] {
		return fmt.Errorf("invalid image type detected: %s", detectedType)
	}

	return nil
}

// detectImageType detects the MIME type from file content
func detectImageType(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	// Check for JPEG
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// Check for PNG
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return "image/png"
	}

	// Check for GIF
	if len(data) >= 6 && string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a" {
		return "image/gif"
	}

	// Check for WebP
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return ""
}
