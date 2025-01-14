package logger

import (
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
)

var loggerCache = make(map[string]*logrus.Logger)
var mutex = new(sync.Mutex)

func Logger(target any) *logrus.Logger {
	mutex.Lock()
	defer mutex.Unlock()

	typeName := handlerName(target)
	if logger, ok := loggerCache[typeName]; ok {
		return logger
	}

	logger := logrus.New()
	logger.SetLevel(logrus.StandardLogger().Level)
	logger.SetFormatter(namedLogger{
		name:      typeName,
		formatter: logrus.StandardLogger().Formatter,
	})
	loggerCache[typeName] = logger

	return logger
}

func handlerName(handler any) string {
	var name string
	t := reflect.TypeOf(handler)
	if t.Kind() == reflect.Ptr {
		name = t.Elem().Name()
	} else {
		name = t.Name()
	}

	return name
}

type namedLogger struct {
	name      string
	formatter logrus.Formatter
}

func (d namedLogger) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Data["name"] = d.name
	return d.formatter.Format(entry)
}
