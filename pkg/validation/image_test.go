package validation

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

// Helper function to create a test file header with specific content and MIME type
func createTestFile(t *testing.T, filename string, content []byte, contentType string) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	// Write content
	if _, err := part.Write(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	// Must close writer before reading body
	contentTypeHeader := writer.FormDataContentType()
	writer.Close()

	// Create HTTP request with multipart form
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", contentTypeHeader)

	// Parse multipart form
	if err := req.ParseMultipartForm(10 << 20); err != nil {
		t.Fatalf("Failed to parse multipart form: %v", err)
	}

	// Get file header
	_, fileHeader, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("Failed to get form file: %v", err)
	}

	// Set Content-Type header if specified
	if contentType != "" {
		fileHeader.Header.Set("Content-Type", contentType)
	}

	return fileHeader
}

// JPEG magic bytes
var jpegMagicBytes = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}

// PNG magic bytes
var pngMagicBytes = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// GIF magic bytes
var gifMagicBytes = []byte("GIF89a")

// WebP magic bytes (RIFF header + WEBP)
var webpMagicBytes = []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}

// PDF magic bytes (for testing invalid file types)
var pdfMagicBytes = []byte("%PDF-1.4")

func TestValidateImageFile_ValidImages(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     []byte
		contentType string
	}{
		{
			name:        "Valid JPEG",
			filename:    "test.jpg",
			content:     jpegMagicBytes,
			contentType: "image/jpeg",
		},
		{
			name:        "Valid PNG",
			filename:    "test.png",
			content:     pngMagicBytes,
			contentType: "image/png",
		},
		{
			name:        "Valid GIF",
			filename:    "test.gif",
			content:     gifMagicBytes,
			contentType: "image/gif",
		},
		{
			name:        "Valid WebP",
			filename:    "test.webp",
			content:     webpMagicBytes,
			contentType: "image/webp",
		},
		{
			name:        "JPEG with .jpeg extension",
			filename:    "test.jpeg",
			content:     jpegMagicBytes,
			contentType: "image/jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileHeader := createTestFile(t, tt.filename, tt.content, tt.contentType)
			err := ValidateImageFile(fileHeader)
			if err != nil {
				t.Errorf("ValidateImageFile() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateImageFile_InvalidExtensions(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  []byte
	}{
		{
			name:     "Invalid extension .txt",
			filename: "test.txt",
			content:  []byte("not an image"),
		},
		{
			name:     "Invalid extension .pdf",
			filename: "test.pdf",
			content:  pdfMagicBytes,
		},
		{
			name:     "Invalid extension .doc",
			filename: "test.doc",
			content:  []byte("document"),
		},
		{
			name:     "No extension",
			filename: "test",
			content:  jpegMagicBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileHeader := createTestFile(t, tt.filename, tt.content, "")
			err := ValidateImageFile(fileHeader)
			if err == nil {
				t.Errorf("ValidateImageFile() expected error for invalid extension, got nil")
			}
		})
	}
}

func TestValidateImageFile_FileSizeValidation(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{
			name:    "File size within limit (1MB)",
			size:    1 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "File size at limit (10MB)",
			size:    10 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "File size exceeds limit (11MB)",
			size:    11 * 1024 * 1024,
			wantErr: true,
		},
		{
			name:    "Empty file",
			size:    0,
			wantErr: true,
		},
		{
			name:    "Small valid file (100KB)",
			size:    100 * 1024,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create content of specified size
			content := make([]byte, tt.size)
			// Add JPEG magic bytes at the beginning
			if tt.size > 0 {
				copy(content, jpegMagicBytes)
			}

			fileHeader := createTestFile(t, "test.jpg", content, "image/jpeg")
			err := ValidateImageFile(fileHeader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImageFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateImageFile_InvalidMIMETypes(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     []byte
		contentType string
	}{
		{
			name:        "PDF file with wrong extension",
			filename:    "test.jpg",
			content:     pdfMagicBytes,
			contentType: "application/pdf",
		},
		{
			name:        "Text file disguised as image",
			filename:    "test.png",
			content:     []byte("This is not an image"),
			contentType: "text/plain",
		},
		{
			name:        "JPEG content but wrong MIME type",
			filename:    "test.jpg",
			content:     jpegMagicBytes,
			contentType: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileHeader := createTestFile(t, tt.filename, tt.content, tt.contentType)
			err := ValidateImageFile(fileHeader)
			if err == nil {
				t.Errorf("ValidateImageFile() expected error for invalid MIME type, got nil")
			}
		})
	}
}

func TestValidateImageFile_NilFile(t *testing.T) {
	err := ValidateImageFile(nil)
	if err == nil {
		t.Errorf("ValidateImageFile() expected error for nil file, got nil")
	}
	if err.Error() != "no file provided" {
		t.Errorf("ValidateImageFile() error = %v, want 'no file provided'", err)
	}
}

func TestDetectImageType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantType string
	}{
		{
			name:     "JPEG detection",
			data:     jpegMagicBytes,
			wantType: "image/jpeg",
		},
		{
			name:     "PNG detection",
			data:     pngMagicBytes,
			wantType: "image/png",
		},
		{
			name:     "GIF detection",
			data:     gifMagicBytes,
			wantType: "image/gif",
		},
		{
			name:     "WebP detection",
			data:     webpMagicBytes,
			wantType: "image/webp",
		},
		{
			name:     "Unknown type",
			data:     []byte("random data"),
			wantType: "",
		},
		{
			name:     "Too short data",
			data:     []byte{0xFF},
			wantType: "",
		},
		{
			name:     "Empty data",
			data:     []byte{},
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType := detectImageType(tt.data)
			if gotType != tt.wantType {
				t.Errorf("detectImageType() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestValidateImageFile_ContentTypeDetection(t *testing.T) {
	// Test that content-type is detected from actual file content, not just headers
	tests := []struct {
		name         string
		filename     string
		actualBytes  []byte
		headerType   string
		shouldPass   bool
		description  string
	}{
		{
			name:        "JPEG content with correct header",
			filename:    "test.jpg",
			actualBytes: jpegMagicBytes,
			headerType:  "image/jpeg",
			shouldPass:  true,
			description: "Valid JPEG should pass",
		},
		{
			name:        "Text content claiming to be JPEG",
			filename:    "test.jpg",
			actualBytes: []byte("This is text, not an image"),
			headerType:  "image/jpeg",
			shouldPass:  false,
			description: "Should detect non-image content",
		},
		{
			name:        "PNG with JPEG extension",
			filename:    "test.jpg",
			actualBytes: pngMagicBytes,
			headerType:  "image/jpeg",
			shouldPass:  false,
			description: "Should detect mismatch between extension and content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileHeader := createTestFile(t, tt.filename, tt.actualBytes, tt.headerType)
			err := ValidateImageFile(fileHeader)

			if tt.shouldPass && err != nil {
				t.Errorf("%s: got error %v, want nil", tt.description, err)
			}
			if !tt.shouldPass && err == nil {
				t.Errorf("%s: got nil, want error", tt.description)
			}
		})
	}
}

func createLargeFile(t *testing.T, sizeInMB int) *multipart.FileHeader {
	t.Helper()

	size := sizeInMB * 1024 * 1024
	content := make([]byte, size)
	// Add JPEG magic bytes at the beginning
	copy(content, jpegMagicBytes)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "large.jpg")

	// Write in chunks to avoid memory issues
	chunkSize := 1024 * 1024 // 1MB chunks
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		part.Write(content[i:end])
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ParseMultipartForm(32 << 20) // 32MB max memory

	_, fileHeader, _ := req.FormFile("file")
	fileHeader.Header.Set("Content-Type", "image/jpeg")

	return fileHeader
}

func TestValidateImageFile_LargeFile(t *testing.T) {
	// Test with an 11MB file (exceeds 10MB limit)
	t.Run("File exceeds size limit", func(t *testing.T) {
		fileHeader := createLargeFile(t, 11)
		err := ValidateImageFile(fileHeader)
		if err == nil {
			t.Error("Expected error for file exceeding 10MB limit, got nil")
		}
	})

	// Test with a 9MB file (within limit)
	t.Run("File within size limit", func(t *testing.T) {
		fileHeader := createLargeFile(t, 9)
		err := ValidateImageFile(fileHeader)
		if err != nil {
			t.Errorf("Expected nil for file within limit, got error: %v", err)
		}
	})
}

func TestAllowedImageTypes(t *testing.T) {
	// Ensure all expected types are allowed
	expectedTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
		"image/gif",
	}

	for _, contentType := range expectedTypes {
		if !AllowedImageTypes[contentType] {
			t.Errorf("Expected %s to be allowed, but it's not in AllowedImageTypes", contentType)
		}
	}

	// Ensure unexpected types are not allowed
	unexpectedTypes := []string{
		"image/svg+xml",
		"image/bmp",
		"application/pdf",
		"text/plain",
	}

	for _, contentType := range unexpectedTypes {
		if AllowedImageTypes[contentType] {
			t.Errorf("Expected %s to NOT be allowed, but it is in AllowedImageTypes", contentType)
		}
	}
}

func TestAllowedImageExtensions(t *testing.T) {
	// Ensure all expected extensions are allowed
	expectedExtensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

	for _, ext := range expectedExtensions {
		if !AllowedImageExtensions[ext] {
			t.Errorf("Expected %s to be allowed, but it's not in AllowedImageExtensions", ext)
		}
	}

	// Ensure unexpected extensions are not allowed
	unexpectedExtensions := []string{".svg", ".bmp", ".pdf", ".txt"}

	for _, ext := range unexpectedExtensions {
		if AllowedImageExtensions[ext] {
			t.Errorf("Expected %s to NOT be allowed, but it is in AllowedImageExtensions", ext)
		}
	}
}

// Benchmark tests
func BenchmarkValidateImageFile(b *testing.B) {
	content := make([]byte, 1024*1024) // 1MB
	copy(content, jpegMagicBytes)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.jpg")
	io.Copy(part, bytes.NewReader(content))
	writer.Close()

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ParseMultipartForm(10 << 20)
	_, fileHeader, _ := req.FormFile("file")
	fileHeader.Header.Set("Content-Type", "image/jpeg")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateImageFile(fileHeader)
	}
}
