package logg

import (
	"encoding/json"
	"os"
	"runtime"
	"time"
)

type brush func(string) string

func newBrush(color string) brush {
	pre := "\033["
	reset := "\033[0m"
	return func(text string) string {
		return pre + color + "m" + text + reset
	}
}

var colors = []brush{
	newBrush("1;35"), //LevelFatal
	newBrush("1;31"), //LevelError
	newBrush("1;33"), //LevelWarn
	newBrush("1;34"), //LevelInfo
	newBrush("1;34"), //LevelDebug
}

type consoleWriter struct {
	lg       *logWriter
	Level    int  `json:"level"`
	Colorful bool `json:"color"`
}

//NewConsoleAppender create a console appender
func newConsoleAppender() Appender {
	w := &consoleWriter{
		lg:       newLogWriter(os.Stdout),
		Level:    LevelDebug,
		Colorful: runtime.GOOS != "windows",
	}
	return w
}

//Init config like `{"level":1}`
func (c *consoleWriter) Init(config string) error {
	if len(config) == 0 {
		return nil
	}
	err := json.Unmarshal([]byte(config), c)
	if runtime.GOOS == "windows" {
		c.Colorful = false
	}
	return err
}

func (c *consoleWriter) WriteMsg(when time.Time, msg string, level int) error {
	if level > c.Level {
		return nil
	}
	if c.Colorful {
		msg = colors[level](msg)
	}
	c.lg.println(when, msg)
	return nil
}

func (c *consoleWriter) Flush() {

}

func (c *consoleWriter) Destroy() {

}

func init() {
	RegisterAppender("console", newConsoleAppender)
}
