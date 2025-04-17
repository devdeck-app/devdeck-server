package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LogLevel defines the severity level of log messages
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	// Logger instances for different log levels
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	WarnLogger  *log.Logger
	ErrorLogger *log.Logger
	FatalLogger *log.Logger

	// Current log level
	currentLogLevel LogLevel = INFO

	// Log file
	logFile *os.File
)

// SetupLogging initializes the logging system with appropriate output
func SetupLogging(logToFile bool, level LogLevel) {
	// Set the current log level
	currentLogLevel = level

	// Common log flags
	logFlags := log.Ldate | log.Ltime | log.Lshortfile

	// Default to stdout/stderr
	debugWriter := io.Writer(os.Stdout)
	infoWriter := io.Writer(os.Stdout)
	warnWriter := io.Writer(os.Stdout)
	errorWriter := io.Writer(os.Stderr)
	fatalWriter := io.Writer(os.Stderr)

	// If logging to file is requested
	if logToFile {
		// Create logs directory if it doesn't exist
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Error getting user home directory: %v", err)
		}

		logDir := filepath.Join(home, ".config", "devdeck", "logs")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Fatalf("Error creating log directory: %v", err)
		}

		// Create log file with timestamp
		timestamp := time.Now().Format("2006-01-02")
		logPath := filepath.Join(logDir, fmt.Sprintf("devdeck-%s.log", timestamp))
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening log file: %v", err)
		}

		// Store for closing later
		logFile = file

		// Set output to both console and file for all loggers
		debugWriter = io.MultiWriter(os.Stdout, file)
		infoWriter = io.MultiWriter(os.Stdout, file)
		warnWriter = io.MultiWriter(os.Stdout, file)
		errorWriter = io.MultiWriter(os.Stderr, file)
		fatalWriter = io.MultiWriter(os.Stderr, file)

		log.Printf("Logging to file: %s", logPath)
	}

	// Initialize loggers with appropriate prefixes and outputs
	DebugLogger = log.New(debugWriter, "DEBUG: ", logFlags)
	InfoLogger = log.New(infoWriter, "INFO: ", logFlags)
	WarnLogger = log.New(warnWriter, "WARN: ", logFlags)
	ErrorLogger = log.New(errorWriter, "ERROR: ", logFlags)
	FatalLogger = log.New(fatalWriter, "FATAL: ", logFlags)
}

// CloseLogger closes any open log files
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

// Debug logs a debug message if debug level is enabled
func Debug(format string, v ...any) {
	if currentLogLevel <= DEBUG {
		DebugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Info logs an informational message
func Info(format string, v ...any) {
	if currentLogLevel <= INFO {
		InfoLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Warn logs a warning message
func Warn(format string, v ...any) {
	if currentLogLevel <= WARN {
		WarnLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Error logs an error message
func Error(format string, v ...any) {
	if currentLogLevel <= ERROR {
		ErrorLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Fatal logs a fatal message and exits the application
func Fatal(format string, v ...any) {
	if currentLogLevel <= FATAL {
		FatalLogger.Output(2, fmt.Sprintf(format, v...))
	}
	os.Exit(1)
}

// GetLogLevel returns a log level from a string
func GetLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return INFO
	}
}

