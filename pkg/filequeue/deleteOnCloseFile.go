package filequeue

import "os"

type DeleteOnCloseFile struct {
	*os.File
}

func NewDeleteOnCloseFile(f *os.File) *DeleteOnCloseFile {
	return &DeleteOnCloseFile{File: f}
}

func (d *DeleteOnCloseFile) Close() error {
	_ = d.File.Close()
	return os.Remove(d.File.Name())
}
