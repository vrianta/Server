package gulog

import (
	"fmt"

	gonfig "github.com/vrianta/gonfig/v1"
)

// global config
var conf = gonfig.New[Config](true)

// logLevel enum
type _type int

const (
	Unknown _type = iota
	DEBUG
	INFO
	WARN
	ERROR
)

var logLevelString = [...]string{
	Unknown: "Unknown",
	DEBUG:   "DEBUG",
	INFO:    "INFO",
	WARN:    "WARN",
	ERROR:   "ERROR",
}

var colors = [...]string{
	Unknown: "\033[0m",
	DEBUG:   "\033[33m",
	INFO:    "\033[32m",
	WARN:    "\033[33m",
	ERROR:   "\033[31m",
}

func Success(msg string, args ...any) {
	log(INFO, &conf, msg, args...)
}

// Colored log wrappers
func Debug(msg string, args ...any) {
	log(DEBUG, &conf, msg, args...)
}
func Info(msg string, args ...any) {
	log(INFO, &conf, msg, args...)
}
func Warn(msg string, args ...any) {
	log(WARN, &conf, msg, args...)
}
func Error(msg string, args ...any) {
	log(ERROR, &conf, msg, args...)
}

// Legacy support
func WriteLog(messages ...any) {
	fmt.Println(messages...)
}

// Laggacy Support
func WriteLogf(msg string, args ...any) {

	fmt.Printf(msg, args...)
}
