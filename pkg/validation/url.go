package validation

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// MaxURLLength is the maximum allowed URL length (500 characters)
	MaxURLLength = 500
)

// ValidateURL validates a URL string
func ValidateURL(urlString string) error {
	// Trim whitespace
	urlString = strings.TrimSpace(urlString)

	// Empty strings are allowed (optional field)
	if urlString == "" {
		return nil
	}

	// Check maximum length
	if len(urlString) > MaxURLLength {
		return fmt.Errorf("URL must be less than %d characters (current length: %d)", MaxURLLength, len(urlString))
	}

	// Parse the URL
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Require HTTPS protocol for security
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use HTTPS protocol (current: %s)", parsedURL.Scheme)
	}

	// Ensure host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	return nil
}
