package server

import (
	"reflect"
	"sync"
	"time"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/sirupsen/logrus"
)

var daemonScanWait = 5 * time.Second

type DaemonHandler interface {
	Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) error
	Close() error
}

type QueueFactory func(name string) filequeue.Queue
type Daemon struct {
	closeRequest chan struct{}
	handlers     []DaemonHandler
	wgClosed     *sync.WaitGroup
	queueFactory QueueFactory
}

type daemonLogger struct {
	daemonName string
	formatter  logrus.Formatter
}

func (d daemonLogger) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Data["daemon"] = d.daemonName
	return d.formatter.Format(entry)
}

func NewDaemon(queueFactory QueueFactory, handlers []DaemonHandler) *Daemon {
	return &Daemon{
		handlers: handlers,
		wgClosed: new(sync.WaitGroup),
	}
}

func (d *Daemon) Start() error {
	d.closeRequest = make(chan struct{})
	d.wgClosed.Add(len(d.handlers))

	outputQueues := make([]filequeue.Queue, len(d.handlers)-1)
	for i := 0; i < len(d.handlers)-1; i++ {
		outputQueues[i] = d.queueFactory(handlerName(d.handlers[i]))
	}

	for i, handler := range d.handlers {
		var inputQueue, outputQueue filequeue.Queue
		if i > 0 {
			inputQueue = outputQueues[i-1]
		}
		if i < len(d.handlers)-1 {
			outputQueue = outputQueues[i]
		}

		go d.run(handler, inputQueue, outputQueue)
	}
	return nil
}

func (d *Daemon) Stop() error {
	close(d.closeRequest)
	d.wgClosed.Wait()
	return nil
}

func (d *Daemon) run(handler DaemonHandler, inputQueue, outputQueue filequeue.Queue) {
	for {
		select {
		case <-d.closeRequest:
			handler.Close()
			d.wgClosed.Done()
			return
		case <-time.After(daemonScanWait):
		}

		var inputFile filequeue.QueueFile
		if inputQueue != nil {
			if p, err := inputQueue.Dequeue(); err == nil {
				inputFile = p
				defer p.Close()
			} else {
				logrus.WithError(err).Error("Failed to dequeue")
			}
		}

		if inputQueue != nil && inputFile == nil {
			continue
		}

		d.runHandler(handler, inputFile, outputQueue)
	}
}

func (d *Daemon) runHandler(handler DaemonHandler, inputFile filequeue.QueueFile, outputQueue filequeue.Queue) {
	typeName := handlerName(handler)

	logger := logrus.New()
	logger.SetFormatter(daemonLogger{
		daemonName: typeName,
		formatter:  logrus.StandardLogger().Formatter,
	})

	defer func() {
		if r := recover(); r != nil {
			logger.WithField("recover", r).Error("Recovered")
		}
	}()

	err := handler.Run(logger, inputFile, outputQueue)
	if err != nil {
		logger.WithError(err).Error("Failed to run handler")
	} else if inputFile != nil {
		inputFile.Done()
	}
}

func handlerName(handler DaemonHandler) string {
	var name string
	t := reflect.TypeOf(handler)
	if t.Kind() == reflect.Ptr {
		name = t.Elem().Name()
	} else {
		name = t.Name()
	}

	return name
}
