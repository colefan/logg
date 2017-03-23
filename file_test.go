package logg

import "testing"

func TestFileAppender(t *testing.T) {
	log := NewLogger(100)
	log.SetAppender("file", `{"filename":"test.log","level":4}`)
	log.SetAppender("console", `{"level":4}`)
	log.Async()
	log.Debug("I am debug file")
	log.Info("I am info file")
	log.Warn("I am warn file")
	log.Error("I am error file")
	log.Flush()
	log.Close()

}
