package server

import (
	"reflect"
	"sync"
	"time"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/logger"
	"github.com/sirupsen/logrus"
)

var log = logger.Logger(Daemon{})
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

func NewDaemon(queueFactory QueueFactory, handlers []DaemonHandler) *Daemon {
	return &Daemon{
		handlers:     handlers,
		wgClosed:     new(sync.WaitGroup),
		queueFactory: queueFactory,
	}
}

func (d *Daemon) Start() error {
	log.Debug("Starting daemon")
	d.closeRequest = make(chan struct{})
	d.wgClosed.Add(len(d.handlers))

	log.Debug("Starting handlers")
	outputQueues := make([]filequeue.Queue, len(d.handlers)-1)
	for i := 0; i < len(d.handlers)-1; i++ {
		outputQueues[i] = d.queueFactory(handlerName(d.handlers[i]))
	}
	log.Debugf("Output queues: %v", outputQueues)

	for i, handler := range d.handlers {
		var inputQueue, outputQueue filequeue.Queue
		if i > 0 {
			inputQueue = outputQueues[i-1]
		}
		if i < len(d.handlers)-1 {
			outputQueue = outputQueues[i]
		}

		logrus.WithField("handler", handlerName(handler)).Debug("Starting handler")
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
	handlerLogger := logger.Logger(handler)

	for {
		select {
		case <-d.closeRequest:
			handlerLogger.Debug("Closing handler")
			handler.Close()
			d.wgClosed.Done()
			return
		case <-time.After(daemonScanWait):
			handlerLogger.Debug("Running handler")
		}

		var inputFile filequeue.QueueFile
		if inputQueue != nil {
			if p, err := inputQueue.Dequeue(); err == nil {
				if p == nil {
					handlerLogger.Debug("No files in inputQueue")
					continue
				}
				handlerLogger.Debug("Dequeued from inputQueue")
				inputFile = p
				defer p.Close()
			} else {
				logrus.WithError(err).Error("Failed to dequeue")
			}
		}

		if inputQueue != nil && inputFile == nil {
			handlerLogger.Debug("No files in inputQueue")
			continue
		}

		d.runHandler(handler, inputFile, outputQueue)
	}
}

func (d *Daemon) runHandler(handler DaemonHandler, inputFile filequeue.QueueFile, outputQueue filequeue.Queue) {
	handlerLogger := logger.Logger(handler)

	defer func() {
		if r := recover(); r != nil {
			handlerLogger.WithField("recover", r).Error("Recovered")
		}
	}()

	err := handler.Run(handlerLogger, inputFile, outputQueue)
	if err != nil {
		handlerLogger.WithError(err).Error("Failed to run handler")
	} else if inputFile != nil {
		inputFile.Done()
	}
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
