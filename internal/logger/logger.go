package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ANSI colors for console
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
)

// colorLevelEncoder encodes level with colors: Error=red, Warn=yellow, Info=green, Debug=cyan.
func colorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var s string
	switch l {
	case zapcore.DebugLevel:
		s = colorCyan + "DEBUG" + colorReset
	case zapcore.InfoLevel:
		s = colorGreen + "INFO " + colorReset
	case zapcore.WarnLevel:
		s = colorYellow + "WARN " + colorReset
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		s = colorRed + "ERROR" + colorReset
	default:
		s = l.CapitalString()
	}
	enc.AppendString(s)
}

// NewDevelopment returns a zap.Logger with colored console output for local dev.
func NewDevelopment() *zap.Logger {
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = colorLevelEncoder
	enc := zapcore.NewConsoleEncoder(cfg)
	core := zapcore.NewCore(enc, zapcore.AddSync(os.Stderr), zapcore.DebugLevel)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}
