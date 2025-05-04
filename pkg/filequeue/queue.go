package filequeue

import (
	"io"
	"os"
)

var baseDir = os.TempDir() + "/scanner-tool-queue"

type Queue interface {
	Enqueue(data []byte) error
	EnqueueFilePath(existingFilePath string) error
	Dequeue() (QueueFile, error)
}

type QueueFile interface {
	io.Reader
	io.ReaderAt
	io.Closer

	Done()
	Size() (int64, error)
}
