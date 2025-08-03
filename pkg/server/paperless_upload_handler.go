package server

import (
	"bytes"
	"io"
	"strings"

	"github.com/schidstorm/scanner-tool/pkg/paperless"
	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/sirupsen/logrus"
)

type PaperlessUploadHandler struct {
	paperless *paperless.Paperless
}

func (u *PaperlessUploadHandler) WithPaperless(paperless *paperless.Paperless) *PaperlessUploadHandler {
	u.paperless = paperless
	return u
}

func (u *PaperlessUploadHandler) Run(logger *logrus.Logger, input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) (resErr error) {
	for f := range input {
		rc, err := f.Open()
		if err != nil {
			logrus.Errorf("Failed to open file %s: %v", f.FileInfo().Name(), err)
			return err
		}
		defer rc.Close()

		pdfData, err := io.ReadAll(rc)
		if err != nil {
			logrus.Errorf("Failed to read file: %v", err)
			return err
		}

		fileReader := bytes.NewReader(pdfData)

		err = u.paperless.Upload(fileReader, paperless.UploadOptions{
			Title: f.FileInfo().Name(),
			Tags:  strings.Split(f.Metadata()["tags"], ","),
		})

		if err != nil {
			logrus.Errorf("Failed to upload file to Paperless: %v", err)
			return err
		}
	}

	return nil
}

func (u *PaperlessUploadHandler) Close() error {
	return nil
}
