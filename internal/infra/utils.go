package infra

import (
	"gorm.io/gorm/logger"
	"strings"
)

// ParseGormLevel takes a string level and returns the gorm Logger log level constant.
func ParseGormLevel(lvl string) logger.LogLevel {
	switch strings.ToLower(lvl) {
	case "panic", "fatal", "info", "debug", "trace":
		return logger.Info
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	default:
		return logger.Silent
	}
}
