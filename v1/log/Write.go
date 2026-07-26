package gulog

// this is global log writing
import (
	gonfig "github.com/vrianta/gonfig/v1"
)

type writer struct {
	conf Config
}
type Config struct {
	// Stop the Log Printing in the console useful for production environment
	NoLog bool `env:"NO_GULOG" arg:"no-gulog" default:"false"`
	// it let the user decide how he want to see the log supported formats are only json and plain texts now
	Format string `env:"GULOG_FORMAT" arg:"gulog-format" default:"text"`
	// a file name where user want to store the log, by default it is empty meens user do not want to create logs
	File string `env:"GULOG_LOGFIE" arg:"gulog-logfile" default:""`
}

func New(_conf *Config) writer {
	if _conf == nil {
		return writer{
			conf: gonfig.New[Config](false),
		}
	}

	return writer{
		conf: *_conf,
	}
}

func (w *writer) Success(msg string, args ...any) {
	log(INFO, &w.conf, msg, args...)
}

// Colored log wrappers
func (w *writer) Debug(msg string, args ...any) {
	log(DEBUG, &w.conf, msg, args...)
}
func (w *writer) Info(msg string, args ...any) {
	log(INFO, &w.conf, msg, args...)
}
func (w *writer) Warn(msg string, args ...any) {
	log(WARN, &w.conf, msg, args...)
}
func (w *writer) Error(msg string, args ...any) {
	log(ERROR, &w.conf, msg, args...)
}
