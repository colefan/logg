package logg

import (
	"testing"
)

func TestConsoleAppender(t *testing.T) {
	log := NewLogger(128)
	log.EnableFuncCallDepath(true)
	log.SetAppender("console", `{"level":4}`)
	log.Async()
	log.Debug("hello i am debug")
	log.Info("hello i am info")
	log.Warn("hello i am warning")
	log.Error("hello i am error")
	log.Fatal("hello i am fatal")
	log.Flush()
	log.Close()
}
