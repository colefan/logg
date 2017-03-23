package logg

import (
	"testing"
)

func TestLog(t *testing.T) {
	log := NewLogger(100)
	log.LoadConfig("log_config.ini")
	log.Async()
	log.Debug("I am debug log")
	log.Info("I am info log")
	log.Warn("I am warn log")
	log.Error("I am error log")
	log.Flush()
	log.Close()
}
