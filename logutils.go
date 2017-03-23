package logg

import (
	"io"
	"sync"
	"time"
)

type logWriter struct {
	sync.Mutex
	writer io.Writer
}

func newLogWriter(wr io.Writer) *logWriter {
	return &logWriter{writer: wr}
}

func (lg *logWriter) println(when time.Time, msg string) {
	lg.Lock()
	str := when.Format("2006-01-02 15:03:04")
	str = str + msg + "\n"
	lg.writer.Write([]byte(str))
	lg.Unlock()
}
