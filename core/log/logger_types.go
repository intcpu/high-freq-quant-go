package log

import (
	"io"
	"sync"
)

const (
	TimestampFormat = " 2006-01-02 15:04:05.000 "
	Spacer          = " | "
	// DefaultMaxFileSize for logger rotation file
	DefaultMaxFileSize int64 = 100
)

var (
	logger = &Logger{}
	// FileLoggingConfiguredCorrectly flag set during config check if file logging meets requirements
	FileLoggingConfiguredCorrectly bool
	// GlobalLogConfig holds global configuration options for logger
	GlobalLogConfig = &Config{}
	// GlobalLogFile hold global configuration options for file logger
	GlobalLogFile = &Rotate{}

	eventPool = &sync.Pool{
		New: func() interface{} {
			return &Event{
				data: make([]byte, 0, 80),
			}
		},
	}

	// LogPath system path to store log files in
	LogPath string

	// RWM read/write mutex for logger
	RWM = &sync.RWMutex{}
)

// Config holds configuration settings loaded from bot config
type Config struct {
	Enabled *bool `json:"enabled"`
	SubLoggerConfig
	LoggerFileConfig *LoggerFileConfig `json:"fileSettings,omitempty"`
	AdvancedSettings AdvancedSettings  `json:"advancedSettings"`
	SubLoggers       []SubLoggerConfig `json:"subloggers,omitempty"`
}

type AdvancedSettings struct {
	ShowLogSystemName *bool   `json:"showLogSystemName"`
	Spacer            string  `json:"spacer"`
	TimeStampFormat   string  `json:"timeStampFormat"`
	Headers           Headers `json:"headers"`
}

type Headers struct {
	Info  string `json:"info"`
	Warn  string `json:"warn"`
	Debug string `json:"debug"`
	Error string `json:"error"`
}

// SubLoggerConfig holds sub logger configuration settings loaded from bot config
type SubLoggerConfig struct {
	Name   string `json:"name,omitempty"`
	Level  string `json:"level"`
	Output string `json:"output"`
}

type LoggerFileConfig struct {
	FileName string `json:"filename,omitempty"`
	Rotate   *bool  `json:"rotate,omitempty"`
	MaxSize  int64  `json:"maxsize,omitempty"`
}

// Logger each instance of logger settings
type Logger struct {
	ShowLogSystemName                                bool
	Timestamp                                        string
	InfoHeader, ErrorHeader, DebugHeader, WarnHeader string
	Spacer                                           string
}

// Levels flags for each sub logger type
type Levels struct {
	Info, Debug, Warn, Error bool
}

type SubLogger struct {
	name string
	Levels
	output io.Writer
}

// Event holds the data sent to the log and which multiwriter to send to
type Event struct {
	data   []byte
	output io.Writer
}

type multiWriter struct {
	writers []io.Writer
	mu      sync.RWMutex
}
