package server

import (
	"path"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/filesystem"
	"github.com/sirupsen/logrus"
)

type UploadHandler struct {
	cifs *filesystem.Cifs
}

func (u *UploadHandler) WithCifs(cifs *filesystem.Cifs) *UploadHandler {
	u.cifs = cifs

	return u
}

func (u *UploadHandler) Run(logger *logrus.Logger, inputQueue, outputQueue *filequeue.Queue) error {
	file, err := inputQueue.Dequeue()
	if err != nil {
		return err
	}
	if file == nil {
		return nil
	}
	defer file.Close()

	err = u.cifs.UploadReader(path.Base(file.Name())+".pdf", file)
	if err != nil {
		return err
	}

	return nil
}

func (u *UploadHandler) Close() error {
	return nil
}
