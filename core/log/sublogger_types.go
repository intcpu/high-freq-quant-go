package log

// Global vars related to the logger package
var (
	SubLoggers = map[string]*SubLogger{}
	Global     *SubLogger
	Mysql      *SubLogger
	Redis      *SubLogger
	Stt        *SubLogger
	Conn       *SubLogger
	Wss        *SubLogger
	Http       *SubLogger
	Script     *SubLogger
)

type Log struct {
	LogFilePath  string `json:"logFilePath,omitempty" structs:"logFilePath,omitempty"`
	Level        string `json:"level,omitempty" structs:"level,omitempty"`
	Output       string `json:"output,omitempty" structs:"output,omitempty"`
	MaxFileSize  int64  `json:"maxFileSize,omitempty" structs:"maxFileSize,omitempty"`
	MaxFileCount int    `json:"maxFileCount,omitempty" structs:"maxFileCount,omitempty"`
}
