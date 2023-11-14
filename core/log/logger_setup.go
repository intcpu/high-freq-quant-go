package log

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"high-freq-quant-go/adapter/convert"
)

func getWriters(s *SubLoggerConfig) io.Writer {
	mw := MultiWriter()
	m := mw.(*multiWriter)

	outputWriters := strings.Split(s.Output, "|")
	for x := range outputWriters {
		switch outputWriters[x] {
		case "stdout", "console":
			m.Add(os.Stdout)
		case "stderr":
			m.Add(os.Stderr)
		case "file":
			if FileLoggingConfiguredCorrectly {
				m.Add(GlobalLogFile)
			}
		default:
			m.Add(ioutil.Discard)
		}
	}
	return m
}

// GenDefaultSettings return struct with known sane/working logger settings
func GenDefaultSettings() (log Config) {
	log = Config{
		Enabled: convert.BoolPtr(true),
		SubLoggerConfig: SubLoggerConfig{
			Level: "INFO|DEBUG|WARN|ERROR",
			//Level:  "INFO|WARN|ERROR",
			Output: "console|file",
			//Output: "console|file",
		},
		LoggerFileConfig: &LoggerFileConfig{
			FileName: "log.txt",
			Rotate:   convert.BoolPtr(true),
			MaxSize:  200,
		},
		AdvancedSettings: AdvancedSettings{
			ShowLogSystemName: convert.BoolPtr(false),
			Spacer:            Spacer,
			TimeStampFormat:   TimestampFormat,
			Headers: Headers{
				Info:  "[INFO]",
				Warn:  "[WARN]",
				Debug: "[DEBUG]",
				Error: "[ERROR]",
			},
		},
	}
	return
}

func configureSubLogger(logger, levels string, output io.Writer) error {
	found, logPtr := validSubLogger(logger)
	if !found {
		return fmt.Errorf("logger %v not found", logger)
	}

	logPtr.output = output

	logPtr.Levels = splitLevel(levels)
	SubLoggers[logger] = logPtr

	return nil
}

// SetupSubLoggers configure all sub loggers with provided configuration values
func SetupSubLoggers(s []SubLoggerConfig) {
	for x := range s {
		output := getWriters(&s[x])
		err := configureSubLogger(strings.ToUpper(s[x].Name), s[x].Level, output)
		if err != nil {
			continue
		}
	}
}

// SetupGlobalLogger setup the global loggers with the default global config values
func SetupGlobalLogger() {
	RWM.Lock()
	if FileLoggingConfiguredCorrectly {
		GlobalLogFile = &Rotate{
			FileName: GlobalLogConfig.LoggerFileConfig.FileName,
			MaxSize:  GlobalLogConfig.LoggerFileConfig.MaxSize,
			Rotate:   GlobalLogConfig.LoggerFileConfig.Rotate,
		}
	}

	for x := range SubLoggers {
		SubLoggers[x].Levels = splitLevel(GlobalLogConfig.Level)
		SubLoggers[x].output = getWriters(&GlobalLogConfig.SubLoggerConfig)
	}

	logger = newLogger(GlobalLogConfig)
	RWM.Unlock()
}

func splitLevel(level string) (l Levels) {
	enabledLevels := strings.Split(level, "|")
	for x := range enabledLevels {
		switch level := enabledLevels[x]; level {
		case "DEBUG":
			l.Debug = true
		case "INFO":
			l.Info = true
		case "WARN":
			l.Warn = true
		case "ERROR":
			l.Error = true
		}
	}
	return
}

func RegisterNewSubLogger(logger string) *SubLogger {
	temp := SubLogger{
		name:   strings.ToUpper(logger),
		output: os.Stdout,
	}

	temp.Levels = splitLevel("INFO|WARN|DEBUG|ERROR")
	SubLoggers[logger] = &temp

	return &temp
}

//func registerNewSubFileLogger(logger string) *SubLogger {
//	temp := SubLogger{
//		name:   strings.ToUpper(logger),
//		output: os.Stdout,
//	}
//
//	temp.Levels = splitLevel("INFO|WARN|DEBUG|ERROR")
//	SubLoggers[logger] = &temp
//
//	return &temp
//}

func ConfigLog(logcfg Log) {
	if _, err := os.Stat(logcfg.LogFilePath); os.IsNotExist(err) {
		_ = os.MkdirAll(logcfg.LogFilePath, os.ModePerm)
	}
	RWM.Lock()
	GlobalLogConfig = &Config{
		Enabled: convert.BoolPtr(true),
		SubLoggerConfig: SubLoggerConfig{
			Level:  logcfg.Level,  // "INFO|DEBUG|WARN|ERROR"
			Output: logcfg.Output, // "console|file",
		},
		LoggerFileConfig: &LoggerFileConfig{
			FileName: "log.txt",
			Rotate:   convert.BoolPtr(true),
			MaxSize:  logcfg.MaxFileSize, //200,//Mb
		},
		AdvancedSettings: AdvancedSettings{
			ShowLogSystemName: convert.BoolPtr(true),
			Spacer:            Spacer,
			TimeStampFormat:   TimestampFormat,
			Headers: Headers{
				Info:  "[INFO]",
				Warn:  "[WARN]",
				Debug: "[DEBUG]",
				Error: "[ERROR]",
			},
		},
	}
	FileLoggingConfiguredCorrectly = true
	LogPath = logcfg.LogFilePath
	RWM.Unlock()
	SetupGlobalLogger()
	SetupSubLoggers(GlobalLogConfig.SubLoggers)
}

// register all loggers at package init()
func init() {
	Global = RegisterNewSubLogger("LOG")
	Mysql = RegisterNewSubLogger("MYSQL")
	Redis = RegisterNewSubLogger("REDIS")
	Stt = RegisterNewSubLogger("STT")
	Conn = RegisterNewSubLogger("Conn")
	Wss = RegisterNewSubLogger("WSS")
	Http = RegisterNewSubLogger("HTTP")
	Script = RegisterNewSubLogger("SCRIPT")
}
