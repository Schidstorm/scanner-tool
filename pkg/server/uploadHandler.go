package server

import (
	"fmt"
	"time"

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

func (u *UploadHandler) Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) (resErr error) {
	fileName := fmt.Sprintf("%d_%d.pdf", time.Now().Unix(), time.Now().Nanosecond())
	return u.cifs.UploadReader(fileName, file)
}

func (u *UploadHandler) Close() error {
	return nil
}
