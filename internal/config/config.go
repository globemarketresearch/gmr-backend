package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment string
	Port        string
	Database    DatabaseConfig
	Redis       RedisConfig
	Auth        AuthConfig
	RateLimit   RateLimitConfig
	Cloudflare  CloudflareConfig
	Email       EmailConfig
	PayPal      PayPalConfig
	Stripe      StripeConfig
}

type PayPalConfig struct {
	ClientID     string // PAYPAL_CLIENT_ID
	ClientSecret string // PAYPAL_CLIENT_SECRET
	Mode         string // PAYPAL_MODE: "sandbox" or "live"
}

type StripeConfig struct {
	SecretKey     string // STRIPE_SECRET_KEY
	WebhookSecret string // STRIPE_WEBHOOK_SECRET
}

type EmailConfig struct {
	Host      string // SMTP_HOST
	Port      int    // SMTP_PORT (default 587)
	User      string // SMTP_USER
	Password  string // SMTP_PASSWORD
	From      string // SMTP_FROM
	NotifyTo  string // EMAIL_NOTIFICATION_TO
	CC        string // EMAIL_CC
	ClientURL string // CLIENT_URL (e.g. https://globemarketresearch.com)
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	URL      string // Full database URL (optional, overrides other fields)
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type AuthConfig struct {
	JWTSecret            string
	AccessTokenExpiry    time.Duration
	RefreshTokenExpiry   time.Duration
	Issuer               string
}

type RateLimitConfig struct {
	LoginMaxAttempts int
	LoginWindow      time.Duration
}

type CloudflareConfig struct {
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Endpoint        string
	R2Bucket          string
	R2PublicURL       string
}

func Load() *Config {
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		smtpPort = 587
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		redisDB = 0
	}

	// Parse JWT token expiry durations
	accessTokenExpiry := parseDuration(getEnv("JWT_ACCESS_TOKEN_EXPIRY", "15m"))
	refreshTokenExpiry := parseDuration(getEnv("JWT_REFRESH_TOKEN_EXPIRY", "168h"))

	// Parse rate limit window
	rateLimitWindow := parseDuration(getEnv("RATE_LIMIT_LOGIN_WINDOW", "15m"))
	rateLimitMaxAttempts, err := strconv.Atoi(getEnv("RATE_LIMIT_LOGIN_MAX_ATTEMPTS", "5"))
	if err != nil {
		rateLimitMaxAttempts = 5
	}

	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnv("PORT", "8081"),
		Database: DatabaseConfig{
			URL:      os.Getenv("DATABASE_URL"), // Railway provides this
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "healthcare_market"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Auth: AuthConfig{
			JWTSecret:          getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessTokenExpiry:  accessTokenExpiry,
			RefreshTokenExpiry: refreshTokenExpiry,
			Issuer:             getEnv("JWT_ISSUER", "healthcare-market-research-api"),
		},
		RateLimit: RateLimitConfig{
			LoginMaxAttempts: rateLimitMaxAttempts,
			LoginWindow:      rateLimitWindow,
		},
		Cloudflare: CloudflareConfig{
			R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			R2Endpoint:        getEnv("R2_ENDPOINT", ""),
			R2Bucket:          getEnv("R2_BUCKET", ""),
			R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),
		},
		Email: EmailConfig{
			Host:      getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:      smtpPort,
			User:      getEnv("SMTP_USER", ""),
			Password:  getEnv("SMTP_PASSWORD", ""),
			From:      getEnv("SMTP_FROM", ""),
			NotifyTo:  getEnv("EMAIL_NOTIFICATION_TO", ""),
			CC:        os.Getenv("EMAIL_CC"),
			ClientURL: getEnv("CLIENT_URL", "https://globemarketresearch.com"),
		},
		PayPal: PayPalConfig{
			ClientID:     getEnv("PAYPAL_CLIENT_ID", ""),
			ClientSecret: getEnv("PAYPAL_CLIENT_SECRET", ""),
			Mode:         getEnv("PAYPAL_MODE", "sandbox"),
		},
		Stripe: StripeConfig{
			SecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		if defaultValue == "" {
			log.Fatalf("Environment variable %s is required", key)
		}
		return defaultValue
	}
	return value
}

func parseDuration(durationStr string) time.Duration {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Printf("Invalid duration format '%s', using default 15m", durationStr)
		return 15 * time.Minute
	}
	return duration
}
