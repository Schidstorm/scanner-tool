package queueoutputcreator

import (
	"io"
)

type QueueZipFileWriter interface {
	FileCount() int
	OpenFile(fileName string) io.Writer
	AddFile(fileName string, data []byte) QueueZipFileWriter
	AddFileReader(fileName string, r io.Reader) QueueZipFileWriter
	AttachMetadata(fileName string, metadata *Metadata) QueueZipFileWriter
	Finalize() (string, error)
	Error() error
}

type QueueZipFileReader interface {
	GetFile(fileName string) (*ZipFile, error)
	FileNames() []string
}
