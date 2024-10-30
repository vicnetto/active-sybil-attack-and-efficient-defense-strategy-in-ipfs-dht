package logger

import (
	"io"
	"log"
	"os"
	"time"
)

type Logger struct {
	Info  *log.Logger
	Error *log.Logger
}

func InitializeLogger() Logger {
	var logger Logger
	logger.Info = log.New(&writer{os.Stdout, "2006/01/02 15:04:05 "}, "[info] ", 0)
	logger.Error = log.New(&writer{os.Stdout, "2006/01/02 15:04:05 "}, "[error] ", 0)

	return logger
}

type writer struct {
	io.Writer
	timeFormat string
}

func (w writer) Write(b []byte) (n int, err error) {
	return w.Writer.Write(append([]byte(time.Now().Format(w.timeFormat)), b...))
}
