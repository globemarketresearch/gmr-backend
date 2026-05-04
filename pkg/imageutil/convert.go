package imageutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// ConvertToWebP converts any image (JPEG, PNG, GIF, WebP, TIFF, BMP) to WebP at 85% quality.
// Requires the cwebp binary (apt install webp / brew install webp / winget install WebPTools).
func ConvertToWebP(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	tmpIn, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpIn.Name())
	if _, err := tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpIn.Close()

	tmpOutPath := tmpIn.Name() + ".webp"
	defer os.Remove(tmpOutPath)

	var stderr bytes.Buffer
	cmd := exec.Command("cwebp", "-q", "85", "-quiet", tmpIn.Name(), "-o", tmpOutPath)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("webp conversion failed: %w (%s)", err, stderr.String())
	}

	return os.ReadFile(tmpOutPath)
}
