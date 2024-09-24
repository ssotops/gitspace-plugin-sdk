package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

type RateLimitedLogger struct {
	logger      *log.Logger
	fileLogger  *log.Logger
	lastLogTime map[string]time.Time
	logInterval time.Duration
	mu          sync.Mutex
	logFile     *os.File
	logFileName string
}

func NewRateLimitedLogger(logDir, prefix string) (*RateLimitedLogger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	date := time.Now().Format("0102")
	index := 1
	var logFile *os.File
	var err error

	for {
		fileName := fmt.Sprintf("%s_%s_%d.log", prefix, date, index)
		filePath := filepath.Join(logDir, fileName)
		logFile, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
		if os.IsExist(err) {
			index++
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create log file: %w", err)
		}
		break
	}

	fileLogger := log.NewWithOptions(logFile, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           log.DebugLevel,
	})

	consoleLogger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           log.DebugLevel,
	})

	return &RateLimitedLogger{
		logger:      consoleLogger,
		fileLogger:  fileLogger,
		lastLogTime: make(map[string]time.Time),
		logInterval: time.Second * 5,
		logFile:     logFile,
		logFileName: logFile.Name(),
	}, nil
}

func (l *RateLimitedLogger) Log(level log.Level, message string, keyvals ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if lastLog, exists := l.lastLogTime[message]; !exists || now.Sub(lastLog) >= l.logInterval {
		l.logger.Log(level, message, keyvals...)
		l.fileLogger.Log(level, message, keyvals...)
		l.lastLogTime[message] = now
	}
}

func (l *RateLimitedLogger) Debug(message string, keyvals ...interface{}) {
	l.Log(log.DebugLevel, message, keyvals...)
}

func (l *RateLimitedLogger) Info(message string, keyvals ...interface{}) {
	l.Log(log.InfoLevel, message, keyvals...)
}

func (l *RateLimitedLogger) Warn(message string, keyvals ...interface{}) {
	l.Log(log.WarnLevel, message, keyvals...)
}

func (l *RateLimitedLogger) Error(message string, keyvals ...interface{}) {
	l.Log(log.ErrorLevel, message, keyvals...)
}

func (l *RateLimitedLogger) Close() error {
	return l.logFile.Close()
}

func (l *RateLimitedLogger) GetLogFileName() string {
	return l.logFileName
}

func (l *RateLimitedLogger) SetLogLevel(level log.Level) {
	l.logger.SetLevel(level)
	l.fileLogger.SetLevel(level)
}
