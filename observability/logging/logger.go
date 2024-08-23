package logging

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

type Config struct {
	LogLevel     string
	LogToFile    bool
	LogFilePath  string
	LogToConsole bool
	DevMode      bool
}

// NewLogger Write leveled structured logs to either standard err, console or file destination
// In the format: LogTimeStamp Log level Message
func NewLogger(config Config) (*zap.Logger, func(), error) {
	level, err := zapcore.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid log level: %v", err)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core
	var closeFns []func()

	if config.LogToFile {
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		logFile, err := os.OpenFile(config.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("can't open log file: %v", err)
		}
		closeFns = append(closeFns, func() { logFile.Close() })
		writer := zapcore.AddSync(logFile)
		cores = append(cores, zapcore.NewCore(fileEncoder, writer, level))
	}

	if config.LogToConsole {
		var consoleEncoder zapcore.Encoder
		if config.DevMode {
			consoleEncoder = zapcore.NewConsoleEncoder(encoderConfig)
		} else {
			consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
		}
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	}

	// If no logging output is configured, log to stderr as a fallback
	if len(cores) == 0 {
		consoleEncoder := zapcore.NewJSONEncoder(encoderConfig)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stderr), level))
	}

	core := zapcore.NewTee(cores...)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Create a cleanup function
	cleanup := func() {
		// Attempt to sync, but ignore errors
		_ = logger.Sync()
		// Close any open files
		for _, fn := range closeFns {
			fn()
		}
	}

	return logger, cleanup, nil
}
