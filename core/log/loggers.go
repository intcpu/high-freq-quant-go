package log

import (
	"fmt"
	"log"
)

// Info takes a pointer SubLogger struct and string sends to newLogEvent
func Info(sl *SubLogger, data string) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Info {
		return
	}

	displayError(logger.newLogEvent(data, logger.InfoHeader, sl.name, sl.output))
}

// Infoln takes a pointer SubLogger struct and interface sends to newLogEvent
func Infoln(sl *SubLogger, v ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Info {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.InfoHeader, sl.name, sl.output))
}

// Infof takes a pointer SubLogger struct, string & interface formats and sends to Info()
func Infof(sl *SubLogger, data string, v ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Info {
		return
	}

	Info(sl, fmt.Sprintf(data, v...))
}

// Debug takes a pointer SubLogger struct and string sends to multiwriter
func Debug(sl *SubLogger, data string) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Debug {
		return
	}

	displayError(logger.newLogEvent(data, logger.DebugHeader, sl.name, sl.output))
}

// Debugln  takes a pointer SubLogger struct, string and interface sends to newLogEvent
func Debugln(sl *SubLogger, v ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Debug {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.DebugHeader, sl.name, sl.output))
}

// Debugf takes a pointer SubLogger struct, string & interface formats and sends to Info()
func Debugf(sl *SubLogger, data string, v ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Debug {
		return
	}

	Debug(sl, fmt.Sprintf(data, v...))
}

// Warn takes a pointer SubLogger struct & string  and sends to newLogEvent()
func Warn(sl *SubLogger, data string) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Warn {
		return
	}

	displayError(logger.newLogEvent(data, logger.WarnHeader, sl.name, sl.output))
}

// Warnln takes a pointer SubLogger struct & interface formats and sends to newLogEvent()
func Warnln(sl *SubLogger, v ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Warn {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.WarnHeader, sl.name, sl.output))
}

// Warnf takes a pointer SubLogger struct, string & interface formats and sends to Warn()
func Warnf(sl *SubLogger, data string, v ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Warn {
		return
	}

	Warn(sl, fmt.Sprintf(data, v...))
}

// Error takes a pointer SubLogger struct & interface formats and sends to newLogEvent()
func Error(sl *SubLogger, data ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Error {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprint(data...), logger.ErrorHeader, sl.name, sl.output))
}

// Errorln takes a pointer SubLogger struct, string & interface formats and sends to newLogEvent()
func Errorln(sl *SubLogger, v ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Error {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.ErrorHeader, sl.name, sl.output))
}

// Errorf takes a pointer SubLogger struct, string & interface formats and sends to Debug()
func Errorf(sl *SubLogger, data string, v ...interface{}) {
	if sl == nil || !enabled() {
		return
	}

	if !sl.Error {
		return
	}

	Error(sl, fmt.Sprintf(data, v...))
}

func displayError(err error) {
	if err != nil {
		log.Printf("Logger write error: %v\n", err)
	}
}

func enabled() bool {
	RWM.Lock()
	defer RWM.Unlock()
	if GlobalLogConfig == nil || GlobalLogConfig.Enabled == nil {
		return false
	}
	if *GlobalLogConfig.Enabled {
		return true
	}
	return false
}
