package gulog

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"
)

type massage struct {
	data      string
	timestamp string
	level     string
	color     string
	conf      *Config
}

// chan for pushing massages (buffered to avoid blocking the sender)
var massages = make(chan massage, 100)

var waitG = sync.WaitGroup{}

func init() {
	waitG.Add(1)
	go startLogging()
}

func startLogging() {
	Info("gulog: Service has been started, make. \nIf you are using os.exit() or any unprecidented exit in the system please use gulog.Wait() in the main end")
	waitG.Done()
	for {
		msg := <-massages

		// prefer per-message config if provided
		c := conf
		if msg.conf != nil {
			c = *msg.conf
		}

		switch c.Format {
		case "json":
			logEntry := map[string]any{
				"timestamp": msg.timestamp,
				"level":     msg.level,
				"message":   msg.data,
			}
			if data, err := json.Marshal(logEntry); err == nil {
				if !c.NoLog {
					fmt.Print(string(data))
				}
				if c.File != "" {
					if err := os.WriteFile(c.File, []byte(data), fs.ModePerm); err != nil {
						fmt.Errorf("Failed to save the log in %s \nreason: %s", c.File, err.Error())
					}
				}
			}
		case "text":
		default:
			if !c.NoLog {
				fmt.Printf("%s[%s] %s: %s\033[0m\n", msg.color, msg.timestamp, msg.level, msg.data)
			}
			if c.File != "" {
				if err := os.WriteFile(c.File, []byte(msg.data), fs.ModePerm); err != nil {
					fmt.Errorf("Failed to save the log in %s \nreason: %s", c.File, err.Error())
				}
			}
		}

		waitG.Done()

	}
}

func log(level _type, _conf *Config, msg string, args ...any) {

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf(msg, args...)
	waitG.Add(1)
	massages <- massage{
		data:      formatted,
		timestamp: timestamp,
		level:     logLevelString[level],
		color:     colors[level],
		conf:      _conf,
	}
}

func Wait() {
	waitG.Wait()
}
