package logger

import (
	"log/slog"
	"media-crawler-go/internal/config"
	"os"
	"strings"
)

func InitFromConfig() {
	level := parseLevel(config.AppConfig.LogLevel)
	format := strings.ToLower(strings.TrimSpace(config.AppConfig.LogFormat))
	if format == "" {
		format = "json"
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}
	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func Info(msg string, args ...any) {
	slog.Default().Info(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Default().Error(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Default().Warn(msg, args...)
}

func Debug(msg string, args ...any) {
	slog.Default().Debug(msg, args...)
}

func parseLevel(v string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
