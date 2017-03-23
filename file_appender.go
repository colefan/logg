package logg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type fileLogWriter struct {
	sync.Mutex
	Filename   string `json:"filename"`
	fileWriter *os.File
	//Rotate at size
	MaxSize        int `json:"maxsize"`
	maxSizeCurSize int

	//Rotate at daily
	Daily         bool `json:"daily"`
	MaxDays       int  `json:"maxdays"` //日志最长保留时间
	dailyOpenDate int

	Rotate       bool `json:"rotate"`
	Level        int  `json:"Level"`
	fileNameOnly string
	fileSuffix   string
}

func newFileAppender() Appender {
	w := &fileLogWriter{
		Filename: "",
		MaxSize:  0, //0
		Daily:    true,
		MaxDays:  0, //
		Rotate:   true,
		Level:    LevelDebug,
	}
	return w
}

//Init file logger with json config
//json config like:
//{
//"filename":"logs/log.log",
//"maxlines":1000000,
//"maxsize":1<<30,
//"daily":true,
//"maxDays":15,
//"rotate":true,
//}
func (f *fileLogWriter) Init(config string) error {
	err := json.Unmarshal([]byte(config), f)
	if err != nil {
		return err
	}
	if len(f.Filename) == 0 {
		return errors.New("json config must have filename")
	}
	f.fileSuffix = filepath.Ext(f.Filename)
	f.fileNameOnly = strings.TrimSuffix(f.Filename, f.fileSuffix)
	if f.fileSuffix == "" {
		f.fileSuffix = ".log"
	}
	err = f.startLogging()
	return err
}

func (f *fileLogWriter) startLogging() error {
	file, err := f.createLogFile()
	if err != nil {
		return err
	}
	if f.fileWriter != nil {
		f.fileWriter.Close()
	}
	f.fileWriter = file
	return f.initFd()
}

func (f *fileLogWriter) createLogFile() (*os.File, error) {
	fd, err := os.OpenFile(f.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	return fd, err
}

func (f *fileLogWriter) initFd() error {
	fd := f.fileWriter
	fInfo, err := fd.Stat()
	if err != nil {
		return err
	}
	f.maxSizeCurSize = int(fInfo.Size())
	f.dailyOpenDate = time.Now().Day()
	return nil
}

func (f *fileLogWriter) needRotate(size int, day int) bool {
	return (f.MaxSize > 0 && f.maxSizeCurSize >= f.MaxSize) ||
		(f.Daily && day != f.dailyOpenDate)
}

func (f *fileLogWriter) doRotate(logTime time.Time) error {
	_, err := os.Lstat(f.Filename)
	if err != nil {
		return err
	}
	num := 1
	fName := ""
	if f.MaxSize > 0 {
		for ; err == nil && num <= 999; num++ {
			fName = f.fileNameOnly + fmt.Sprintf("_%s_%03d%s", logTime.Format("2006-01-02"), num, f.fileSuffix)
			_, err = os.Lstat(fName)
		}

	} else {
		fName = fmt.Sprintf("%s_%s%s", f.fileNameOnly, logTime.Format("2006-01-02"), f.fileSuffix)
		_, err = os.Lstat(fName)
	}

	if err == nil {
		return errors.New("Rotate: can not find free log number to rename " + f.Filename + "\n")
	}
	f.fileWriter.Close()
	errRename := os.Rename(f.Filename, fName)
	if errRename != nil {
		return errors.New("Rotate: rename error " + errRename.Error())
	}
	errStartLogging := f.startLogging()
	if errStartLogging != nil {
		return errors.New("Rotate: startLogging error " + errStartLogging.Error())
	}
	go f.deleteOldLog()
	return nil
}

func (f *fileLogWriter) deleteOldLog() {
	if f.MaxDays <= 0 {
		return
	}

	dir := filepath.Dir(f.Filename)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Unable to delete old log %s,error %v\n", path, r)
			}
		}()

		if !info.IsDir() && (info.ModTime().Unix() < (time.Now().Unix() - int64(60*60*24*f.MaxDays))) {
			if strings.HasPrefix(filepath.Base(path), f.fileNameOnly) &&
				strings.HasSuffix(filepath.Base(path), f.fileSuffix) {
				os.Remove(path)
			}

		}
		return
	})
}

func (f *fileLogWriter) WriteMsg(when time.Time, msg string, level int) error {
	if level > f.Level {
		return nil
	}
	msg = when.Format("2006-01-02 15:03:04") + msg + "\n"
	if f.Rotate {
		if f.needRotate(len(msg), when.Day()) {
			f.Lock()
			if err := f.doRotate(when); err != nil {
				fmt.Fprintf(os.Stderr, "FileLogAppender %q:%s\n", f.Filename, err.Error())
			}
			f.Unlock()
		}
	}
	f.Lock()
	_, err := f.fileWriter.Write([]byte(msg))
	if err == nil {
		f.maxSizeCurSize += len(msg)
	}
	f.Unlock()
	return err
}

func (f *fileLogWriter) Flush() {
	f.fileWriter.Sync()
}

func (f *fileLogWriter) Destroy() {
	f.fileWriter.Close()
}

func init() {
	RegisterAppender("file", newFileAppender)
}
