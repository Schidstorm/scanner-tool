package queueoutputcreator

import (
	"archive/zip"
	"io"
	"io/fs"
)

type ZipFile struct {
	File     *zip.File
	metadata *Metadata
}

func (z *ZipFile) Open() (io.ReadCloser, error) {
	rc, err := z.File.Open()
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func (z *ZipFile) FileInfo() fs.FileInfo {
	return z.File.FileInfo()
}

func (z *ZipFile) Metadata() map[string]string {
	if z.metadata == nil {
		return make(map[string]string)
	}
	return z.metadata.ToMap()
}
