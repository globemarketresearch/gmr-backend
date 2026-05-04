package validation

import (
	"strings"
	"testing"
)

func TestValidateURL_ValidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "Valid HTTPS LinkedIn URL",
			url:  "https://www.linkedin.com/in/johndoe",
		},
		{
			name: "Valid HTTPS GitHub URL",
			url:  "https://github.com/username",
		},
		{
			name: "Valid HTTPS with path and query params",
			url:  "https://example.com/path/to/resource?param=value",
		},
		{
			name: "Valid HTTPS with fragment",
			url:  "https://example.com/page#section",
		},
		{
			name: "Valid HTTPS with port",
			url:  "https://example.com:8080/path",
		},
		{
			name: "Valid HTTPS subdomain",
			url:  "https://subdomain.example.com/path",
		},
		{
			name: "Valid HTTPS with hyphen in domain",
			url:  "https://my-domain.com/profile",
		},
		{
			name: "Empty string (optional field)",
			url:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if err != nil {
				t.Errorf("ValidateURL() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateURL_InvalidProtocol(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "HTTP instead of HTTPS",
			url:  "http://example.com",
		},
		{
			name: "FTP protocol",
			url:  "ftp://example.com",
		},
		{
			name: "No protocol",
			url:  "example.com",
		},
		{
			name: "File protocol",
			url:  "file:///path/to/file",
		},
		{
			name: "mailto protocol",
			url:  "mailto:user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if err == nil {
				t.Errorf("ValidateURL() expected error for invalid protocol, got nil")
			}
		})
	}
}

func TestValidateURL_MalformedURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "Invalid characters",
			url:  "https://example com/path",
		},
		{
			name: "Missing host",
			url:  "https://",
		},
		{
			name: "Only protocol",
			url:  "https:",
		},
		{
			name: "Invalid URL structure",
			url:  "https:///path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if err == nil {
				t.Errorf("ValidateURL() expected error for malformed URL, got nil")
			}
		})
	}
}

func TestValidateURL_MaxLength(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "URL within limit (100 chars)",
			url:     "https://example.com/" + strings.Repeat("a", 75),
			wantErr: false,
		},
		{
			name:    "URL at limit (500 chars)",
			url:     "https://example.com/" + strings.Repeat("a", 480),
			wantErr: false,
		},
		{
			name:    "URL exceeds limit (501 chars)",
			url:     "https://example.com/" + strings.Repeat("a", 481),
			wantErr: true,
		},
		{
			name:    "URL far exceeds limit (1000 chars)",
			url:     "https://example.com/" + strings.Repeat("a", 980),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "URL with leading whitespace",
			url:     "  https://example.com",
			wantErr: false, // Should be trimmed
		},
		{
			name:    "URL with trailing whitespace",
			url:     "https://example.com  ",
			wantErr: false, // Should be trimmed
		},
		{
			name:    "URL with both leading and trailing whitespace",
			url:     "  https://example.com  ",
			wantErr: false, // Should be trimmed
		},
		{
			name:    "Whitespace only",
			url:     "   ",
			wantErr: false, // Treated as empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_SpecialCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "URL with encoded characters",
			url:     "https://example.com/path%20with%20spaces",
			wantErr: false,
		},
		{
			name:    "URL with international domain",
			url:     "https://例え.jp/path",
			wantErr: false,
		},
		{
			name:    "URL with multiple subdomains",
			url:     "https://sub1.sub2.sub3.example.com",
			wantErr: false,
		},
		{
			name:    "URL with IP address",
			url:     "https://192.168.1.1/path",
			wantErr: false,
		},
		{
			name:    "URL with localhost",
			url:     "https://localhost:8080/path",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "LinkedIn profile URL",
			url:     "https://www.linkedin.com/in/john-doe-123456",
			wantErr: false,
		},
		{
			name:    "Twitter profile URL",
			url:     "https://twitter.com/username",
			wantErr: false,
		},
		{
			name:    "GitHub profile URL",
			url:     "https://github.com/username",
			wantErr: false,
		},
		{
			name:    "Personal website",
			url:     "https://johndoe.com",
			wantErr: false,
		},
		{
			name:    "Blog URL with path",
			url:     "https://medium.com/@username/article-title",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_ErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantErrContain string
	}{
		{
			name:           "HTTP protocol error message",
			url:            "http://example.com",
			wantErrContain: "HTTPS",
		},
		{
			name:           "Length error message",
			url:            "https://example.com/" + strings.Repeat("a", 500),
			wantErrContain: "500 characters",
		},
		{
			name:           "Missing host error",
			url:            "https://",
			wantErrContain: "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if err == nil {
				t.Errorf("ValidateURL() expected error, got nil")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("ValidateURL() error = %v, want error containing %q", err, tt.wantErrContain)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateURL_Valid(b *testing.B) {
	url := "https://www.linkedin.com/in/johndoe"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateURL(url)
	}
}

func BenchmarkValidateURL_Invalid(b *testing.B) {
	url := "http://example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateURL(url)
	}
}

func BenchmarkValidateURL_Empty(b *testing.B) {
	url := ""
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateURL(url)
	}
}
