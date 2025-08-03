package queueoutputcreator

import (
	"bytes"
	"errors"
	"io"
	"os"
)

type MemZipFileCreator struct {
	files map[string]*bytes.Buffer
	err   error
}

func CreateMemZipFileCreator() *MemZipFileCreator {
	return &MemZipFileCreator{
		files: make(map[string]*bytes.Buffer),
	}
}

func (m MemZipFileCreator) Files() map[string]*bytes.Buffer {
	return m.files
}

func (m *MemZipFileCreator) FileCount() int {
	if m.err != nil {
		return 0
	}
	return len(m.files)
}

func (m *MemZipFileCreator) Error() error {
	return m.err
}

func (m *MemZipFileCreator) OpenFile(fileName string) io.Writer {
	if m.err != nil {
		return &nullWriter{}
	}

	if _, exists := m.files[fileName]; exists {
		m.err = os.ErrExist
		return &nullWriter{}
	}

	m.files[fileName] = bytes.NewBuffer(nil)

	return m.files[fileName]
}

func (m *MemZipFileCreator) AddFile(fileName string, data []byte) QueueZipFileWriter {
	if m.err != nil {
		return m
	}

	if _, exists := m.files[fileName]; exists {
		m.err = os.ErrExist
		return m
	}

	m.files[fileName] = bytes.NewBuffer(data)
	return m
}
func (m *MemZipFileCreator) AddFileReader(fileName string, r io.Reader) QueueZipFileWriter {
	if m.err != nil {
		return m
	}

	b := bytes.NewBuffer(nil)
	io.Copy(b, r)
	return m.AddFile(fileName, b.Bytes())
}
func (m *MemZipFileCreator) Finalize() (string, error) {
	return "", errors.New("memZipFileCreator does not support Finalize")
}

func (z *MemZipFileCreator) AttachMetadata(fileName string, metadata *Metadata) QueueZipFileWriter {
	if z.err != nil {
		return z
	}

	if metadata == nil || metadata.IsEmpty() {
		return z
	}

	return z.AddFile(".metadata."+fileName, metadata.Serialize())
}
