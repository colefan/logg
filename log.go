package logg

import (
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/colefan/config"
)

const defautChannelBuffer int = 128

const (
	//LevelFatal define logger level
	LevelFatal = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
)

//Appender logger output interface
type Appender interface {
	Init(config string) error
	WriteMsg(when time.Time, msg string, level int) error
	Destroy()
	Flush()
}

type createAppender func() Appender

var appenderMap = make(map[string]createAppender)

//RegisterAppender register an appender to the logger
func RegisterAppender(name string, appender createAppender) {
	if appender == nil {
		panic("logg: RegisterAppender appender is nil")
	}
	if _, dup := appenderMap[name]; dup {
		panic("logg:RegisterAppender called twice for appender " + name)
	}
	appenderMap[name] = appender
}

type nameAppender struct {
	Appender
	name string
}

type logMsg struct {
	level int
	msg   string
	when  time.Time
}

//BaseLogger struct of logger
type BaseLogger struct {
	lock                sync.Mutex
	level               int
	enableFuncCallDepth bool
	loggerFuncCallDepth int
	msgChan             chan *logMsg
	appenders           []*nameAppender
	async               bool
	logMsgPool          *sync.Pool
	singalChan          chan string
	wg                  sync.WaitGroup
}

//NewLogger create a logger
func NewLogger(channelLen int) *BaseLogger {
	log := new(BaseLogger)
	log.level = LevelDebug
	log.loggerFuncCallDepth = 2
	log.enableFuncCallDepth = false
	log.msgChan = make(chan *logMsg, channelLen)
	log.singalChan = make(chan string, 1)
	log.async = false
	return log
}

//SetAppender
func (log *BaseLogger) SetAppender(appenderName string, config string) error {
	log.lock.Lock()
	defer log.lock.Unlock()
	if appenderName == "console" {
		for _, appender := range log.appenders {
			if appender.name == appenderName {
				return errors.New("logg:duplicate appenderName " + appenderName + " (you have set this appender before)")
			}
		}
	}
	appender, ok := appenderMap[appenderName]
	if !ok {
		return errors.New("logg:unknow appenderName " + appenderName + " (forgotten RegisterAppender?)")
	}
	out := appender()
	err := out.Init(config)
	if err != nil {
		return errors.New("logg: appender init error " + err.Error())
	}
	log.appenders = append(log.appenders, &nameAppender{name: appenderName, Appender: out})
	return nil
}

//Async asynchroonous and start the goroutine
func (log *BaseLogger) Async() *BaseLogger {
	log.async = true
	log.logMsgPool = &sync.Pool{
		New: func() interface{} {
			return &logMsg{}
		},
	}
	log.wg.Add(1)
	go log.startLogging()
	return log
}

func (log *BaseLogger) startLogging() {
	gameOver := false
	for {
		select {
		case msg := <-log.msgChan:
			log.writeToAppender(msg.when, msg.msg, msg.level)
			log.logMsgPool.Put(msg)
		case sg := <-log.singalChan:
			log.flush()
			if sg == "close" {
				for _, out := range log.appenders {
					out.Destroy()
				}
				log.appenders = nil
				gameOver = true
			}
			log.wg.Done()

		}

		if gameOver {
			break
		}

	}

}

func (log *BaseLogger) writeToAppender(time time.Time, msg string, level int) {
	for _, out := range log.appenders {
		err := out.WriteMsg(time, msg, level)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to WriteMsg to appender:%v,error:%v\n", out.name, err)
		}
	}
}

func (log *BaseLogger) writeMsg(level int, msg string) {
	when := time.Now()
	if log.enableFuncCallDepth {
		_, file, line, ok := runtime.Caller(log.loggerFuncCallDepth)
		if !ok {
			file = "???"
			line = 0
		}
		_, filename := path.Split(file)
		msg = msg[0:3] + "[" + filename + ":" + strconv.FormatInt(int64(line), 10) + "]" + msg[3:]

	}

	if log.async {
		m := log.logMsgPool.Get().(*logMsg)
		m.level = level
		m.msg = msg
		m.when = when
		log.msgChan <- m

	} else {
		log.writeToAppender(when, msg, level)
	}
}

//SetLevel setter
func (log *BaseLogger) SetLevel(level int) {
	log.level = level
}

//Level getter
func (log *BaseLogger) Level() int {
	return log.level
}

//SetLogFuncCallDepth setter
func (log *BaseLogger) SetLogFuncCallDepth(d int) {
	log.loggerFuncCallDepth = d
}

//EnableFuncCallDepath setter
func (log *BaseLogger) EnableFuncCallDepath(d bool) {
	log.enableFuncCallDepth = d
}

//Fatal log.Fatal
func (log *BaseLogger) Fatal(format string, v ...interface{}) {
	if LevelFatal > log.level {
		return
	}
	msg := fmt.Sprintf("[F] "+format, v...)
	log.writeMsg(LevelFatal, msg)
}

//Error log.Error
func (log *BaseLogger) Error(format string, v ...interface{}) {
	if LevelError > log.level {
		return
	}
	msg := fmt.Sprintf("[E] "+format, v...)
	log.writeMsg(LevelError, msg)
}

//Warn log.Warn
func (log *BaseLogger) Warn(format string, v ...interface{}) {
	if LevelWarn > log.level {
		return
	}
	msg := fmt.Sprintf("[W] "+format, v...)
	log.writeMsg(LevelWarn, msg)
}

//Info log.Info
func (log *BaseLogger) Info(format string, v ...interface{}) {
	if LevelInfo > log.level {
		return
	}
	msg := fmt.Sprintf("[I] "+format, v...)
	log.writeMsg(LevelInfo, msg)
}

//Debug log.Debug
func (log *BaseLogger) Debug(format string, v ...interface{}) {
	if LevelDebug > log.level {
		return
	}
	msg := fmt.Sprintf("[D] "+format, v...)
	log.writeMsg(LevelDebug, msg)
}

//Flush flush logger's msg
func (log *BaseLogger) Flush() {
	if log.async {
		log.singalChan <- "flush"
		log.wg.Wait()
		log.wg.Add(1)
		return
	}
	log.flush()
}

func (log *BaseLogger) flush() {
	for {
		if len(log.msgChan) > 0 {
			m := <-log.msgChan
			log.writeToAppender(m.when, m.msg, m.level)
			log.logMsgPool.Put(m)
			continue
		}
		break
	}

	for _, out := range log.appenders {
		out.Flush()
	}

}

//Close close the logger
func (log *BaseLogger) Close() {
	if log.async {
		log.singalChan <- "close"
		log.wg.Wait()
	} else {
		log.flush()
		for _, out := range log.appenders {
			out.Destroy()
		}
		log.appenders = nil
	}
	close(log.msgChan)
	close(log.singalChan)
}

func (log *BaseLogger) LoadConfig(filename string) *BaseLogger {
	cnf := config.NewIniConfig()
	err := cnf.Parse(filename)
	if err != nil {
		panic("LoadConfig: Parse filename error " + filename)
	}
	strLevel := cnf.String("logg.root.level")
	if v, ok := levelStrMaps[strLevel]; ok {
		log.SetLevel(v)
	}
	if callFile, err := cnf.Bool("logg.root.callfile"); err == nil {
		log.EnableFuncCallDepath(callFile)
	}

	strStdAppender := cnf.String("logg.appender.stdout")
	if strStdAppender == "console" {
		strL := cnf.String("logg.appender.stdout.level")
		if v, ok := levelStrMaps[strL]; ok {
			log.SetAppender("console", `{"level":`+strconv.Itoa(v)+`}`)
		} else {
			log.SetAppender("console", ``)
		}

	} else if strStdAppender == "file" {
		strConf := `{`
		strFile := cnf.String("logg.appender.stdout.file")
		if len(strFile) > 0 {
			strConf = strConf + `"filename":"` + strFile + `",`
		}
		strL := cnf.String("logg.appender.stdout.level")
		if v, ok := levelStrMaps[strL]; ok {
			strConf = strConf + `"level":` + strconv.Itoa(v) + `,`
		}

		if maxday, err := cnf.Int("logg.appender.stdout.maxday"); err == nil {
			strConf = strConf + `"maxday":` + strconv.Itoa(maxday) + `,`
		}

		if maxsize, err := cnf.Int("logg.appender.stdout.maxsize"); err == nil {
			strConf = strConf + `"maxsize":` + strconv.Itoa(maxsize) + `,`
		}
		if daily, err := cnf.Bool("logg.appender.stdout.daily"); err == nil {
			strConf = strConf + `"daily":` + strconv.FormatBool(daily) + `,`
		}
		if rotate, err := cnf.Bool("logg.appender.stdout.rotate"); err == nil {
			strConf = strConf + `"rotate":` + strconv.FormatBool(rotate) + `,`
		}

		if len(strConf) > len(`{`) {
			strConf = strConf[0:len(strConf)-1] + `}`
		}
		log.SetAppender("file", strConf)
	}

	//other appenders
	appenderList := cnf.Strings("logg.appender")
	for _, name := range appenderList {
		if len(name) == 0 {
			continue
		}

		strPreKey := "logg.appender." + name
		appenderName := cnf.String(strPreKey)
		if appenderName == "console" {
			if v, ok := levelStrMaps[cnf.String(strPreKey+".level")]; ok {
				log.SetAppender("console", `{"level":`+strconv.Itoa(v)+`}`)
			} else {
				log.SetAppender("console", ``)
			}

		} else if appenderName == "file" {
			strConf := `{`
			strFile := cnf.String(strPreKey + ".file")
			if len(strFile) > 0 {
				strConf = strConf + `"filename":"` + strFile + `",`
			}
			strL := cnf.String(strPreKey + ".level")
			if v, ok := levelStrMaps[strL]; ok {
				strConf = strConf + `"level":` + strconv.Itoa(v) + `,`
			}

			if maxday, err := cnf.Int(strPreKey + ".maxday"); err == nil {
				strConf = strConf + `"maxday":` + strconv.Itoa(maxday) + `,`
			}

			if maxsize, err := cnf.Int(strPreKey + ".maxsize"); err == nil {
				strConf = strConf + `"maxsize":` + strconv.Itoa(maxsize) + `,`
			}
			if daily, err := cnf.Bool(strPreKey + ".daily"); err == nil {
				strConf = strConf + `"daily":` + strconv.FormatBool(daily) + `,`
			}
			if rotate, err := cnf.Bool(strPreKey + ".rotate"); err == nil {
				strConf = strConf + `"rotate":` + strconv.FormatBool(rotate) + `,`
			}

			if len(strConf) > len(`{`) {
				strConf = strConf[0:len(strConf)-1] + `}`
			}
			log.SetAppender("file", strConf)

		}

	}

	return log
}

var levelStrMaps = make(map[string]int)

func init() {
	levelStrMaps["debug"] = LevelDebug
	levelStrMaps["info"] = LevelInfo
	levelStrMaps["warn"] = LevelWarn
	levelStrMaps["error"] = LevelError
	levelStrMaps["fatal"] = LevelFatal

}
