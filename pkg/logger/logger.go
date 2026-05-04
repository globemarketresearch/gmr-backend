package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

var Log *slog.Logger

func Init(env string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if env == "development" {
		opts.Level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	Log = slog.New(handler)
}

func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

func LogRequest(c *fiber.Ctx, duration time.Duration) {
	Log.Info("request",
		"method", c.Method(),
		"path", c.Path(),
		"status", c.Response().StatusCode(),
		"duration_ms", duration.Milliseconds(),
		"ip", c.IP(),
		"user_agent", c.Get("User-Agent"),
	)
}
