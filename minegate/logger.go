package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	USE_NONE int = iota
	USE_FILE
	USE_SYSLOG
	USE_BOTH
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var logLevel = INFO

var level_string = [...]string{
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"FATAL",
}

type FileLog struct {
	writer *os.File
	path   string
}

type Message struct {
	message string
	level   Level
}

func NewFdLog(w *os.File) (fl *FileLog) {
	return &FileLog{
		writer: w,
		path:   "",
	}
}

func NewFileLog(f string) (fl *FileLog, err error) {
	w, err := os.OpenFile(f, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return nil, err
	}
	fl = NewFdLog(w)
	fl.path = f
	return
}

var queue = make(chan *Message, 32)
var quit_signal = make(chan byte, 1)

var logger = NewFdLog(os.Stderr)
var prefix = ""
var lock sync.Mutex
var initialized bool

func log_daemon() {
	for {
		select {
		case msg := <-queue:
			lock.Lock()
			if msg.level >= logLevel {
				fmt.Fprintf(logger.writer, "[%5s @ %s] %s%s\n", level_string[msg.level], time.Now().Format("Jan 2 15:04:05.000"), prefix, msg.message)
			}
			if msg.level == FATAL {
				quit_signal <- '\x00'
			}
			lock.Unlock()
		}
	}
}

func LogPrefix(pre string) {
	prefix = pre
}

func LogOpen(f string) (err error) {
	lock.Lock()
	defer lock.Unlock()
	fl, err := NewFileLog(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file %s: %s", f, err)
		return err
	} else {
		logger = fl
		Infof("Log ready.")
	}
	LogInit()
	return nil
}

func LogInit() {
	if !initialized {
		go log_daemon()
		initialized = true
	}
}

func Fatal(msg string) {
	queue <- &Message{
		message: msg,
		level:   FATAL,
	}
	/* Wait for flushing logs. */
	<-quit_signal
	os.Exit(1)
}

func Fatalf(format string, a ...interface{}) {
	Fatal(fmt.Sprintf(format, a...))
}

func Error(msg string) {
	if logLevel > ERROR {
		return
	}
	queue <- &Message{
		message: msg,
		level:   ERROR,
	}
}

func Errorf(format string, a ...interface{}) {
	if logLevel > ERROR {
		return
	}
	queue <- &Message{
		message: fmt.Sprintf(format, a...),
		level:   ERROR,
	}
}

func Warn(msg string) {
	if logLevel > WARN {
		return
	}
	queue <- &Message{
		message: msg,
		level:   WARN,
	}
}

func Warnf(format string, a ...interface{}) {
	if logLevel > WARN {
		return
	}
	queue <- &Message{
		message: fmt.Sprintf(format, a...),
		level:   WARN,
	}
}

func Info(msg string) {
	if logLevel > INFO {
		return
	}
	queue <- &Message{
		message: msg,
		level:   INFO,
	}
}

func Infof(format string, a ...interface{}) {
	if logLevel > INFO {
		return
	}
	queue <- &Message{
		message: fmt.Sprintf(format, a...),
		level:   INFO,
	}
}

func Debug(msg string) {
	if logLevel > DEBUG {
		return
	}
	queue <- &Message{
		message: msg,
		level:   DEBUG,
	}
}

func Debugf(format string, a ...interface{}) {
	if logLevel > DEBUG {
		return
	}
	queue <- &Message{
		message: fmt.Sprintf(format, a...),
		level:   DEBUG,
	}
}

func LogRotate() (err error) {
	lock.Lock()
	defer lock.Unlock()
	logger.writer.Sync() // Ignore error here.
	if logger.path != "" {
		newfd, err := os.OpenFile(logger.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			Errorf("Reopen log file %s: %s", logger.path, err)
			return err
		} else {
			Infof("Reopened log file %s", logger.path)
			newlog := NewFdLog(newfd)
			newlog.path = logger.path
			logger = newlog
		}
	}
	return nil
}
