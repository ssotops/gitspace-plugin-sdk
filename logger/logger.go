package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/ssotops/gitspace-plugin-sdk/gsplug"
)

type RateLimitedLogger struct {
	logger          *log.Logger
	fileLogger      *log.Logger
	lastLogTime     map[string]time.Time
	logInterval     time.Duration
	mu              sync.Mutex
	logFile         *os.File
	logFileName     string
	updatedLogFiles map[string]bool
}

func (l *RateLimitedLogger) GetLogFileName() string {
	return l.logFileName
}

func NewRateLimitedLogger(pluginName string) (*RateLimitedLogger, error) {
	logDir, err := gsplug.GetPluginLogDir(pluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin log directory: %w", err)
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	date := time.Now().Format("20060102")
	index := 0
	var logFile *os.File

	for {
		fileName := fmt.Sprintf("%s_%s_%02d.log", pluginName, date, index)
		filePath := filepath.Join(logDir, fileName)
		file, openErr := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
		if os.IsExist(openErr) {
			index++
			continue
		}
		if openErr != nil {
			return nil, fmt.Errorf("failed to create log file: %w", openErr)
		}
		logFile = file
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
		logger:          consoleLogger,
		fileLogger:      fileLogger,
		lastLogTime:     make(map[string]time.Time),
		logInterval:     time.Second * 5,
		logFile:         logFile,
		logFileName:     logFile.Name(),
		updatedLogFiles: make(map[string]bool),
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
		l.updatedLogFiles[l.logFileName] = true
	}
}

func (l *RateLimitedLogger) Info(message string, keyvals ...interface{}) {
	l.Log(log.InfoLevel, message, keyvals...)
}

func (l *RateLimitedLogger) Debug(message string, keyvals ...interface{}) {
	l.Log(log.DebugLevel, message, keyvals...)
}

func (l *RateLimitedLogger) Error(message string, keyvals ...interface{}) {
	l.Log(log.ErrorLevel, message, keyvals...)
}

func (l *RateLimitedLogger) Warn(message string, keyvals ...interface{}) {
	l.Log(log.WarnLevel, message, keyvals...)
}

func (l *RateLimitedLogger) GetUpdatedLogFiles() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	files := make([]string, 0, len(l.updatedLogFiles))
	for file := range l.updatedLogFiles {
		// Ensure this is the full path
		files = append(files, file)
	}
	return files
}

func (l *RateLimitedLogger) SetLogLevel(level log.Level) {
	l.logger.SetLevel(level)
	l.fileLogger.SetLevel(level)
}
