package server

import (
	"io"
	"io/fs"
	"reflect"
	"sync"
	"time"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/logger"
	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/sirupsen/logrus"
)

var log = logger.Logger(Daemon{})
var daemonScanWait = 5 * time.Second

type InputFile interface {
	Open() (io.ReadCloser, error)
	FileInfo() fs.FileInfo
	Metadata() map[string]string
}

type DaemonHandler interface {
	Run(logger *logrus.Logger, input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) error
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

func (d *Daemon) runHandler(handler DaemonHandler, inputZipFile filequeue.QueueFile, outputQueue filequeue.Queue) {
	handlerLogger := logger.Logger(handler)

	defer func() {
		if r := recover(); r != nil {
			handlerLogger.WithField("recover", r).Error("Recovered")
		}
	}()

	inputFiles := make(chan InputFile)
	go func() {
		defer close(inputFiles)
		if inputZipFile != nil {
			handlerLogger.Debug("Sending inputFile to handler")
			zipReader, err := queueoutputcreator.CreateZipFileReader(inputZipFile)
			if err != nil {
				handlerLogger.WithError(err).Error("Failed to create zip reader")
				inputFiles <- nil
				return
			}

			for _, fileName := range zipReader.FileNames() {
				f, err := zipReader.GetFile(fileName)
				if err != nil {
					handlerLogger.WithError(err).Errorf("Failed to get file %s from zip", fileName)
					continue
				}
				inputFiles <- f
			}
		}
	}()

	outputFiles := queueoutputcreator.CreateZipFileWriter()

	err := handler.Run(handlerLogger, inputFiles, outputFiles)
	if err != nil {
		handlerLogger.WithError(err).Error("Failed to run handler")
	} else if outputFiles.Error() != nil {
		handlerLogger.WithError(outputFiles.Error()).Error("Failed to run handler")
	} else {
		if outputFiles.FileCount() > 0 {
			outputZipPath, _ := outputFiles.Finalize()
			handlerLogger.Debugf("Handler %s created %d files", handlerName(handler), outputFiles.FileCount())
			outputQueue.EnqueueFilePath(outputZipPath)
		}

		if inputZipFile != nil {
			inputZipFile.Done()
		}
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
