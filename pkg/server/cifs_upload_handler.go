package server

import (
	"archive/zip"
	"bytes"
	"os/exec"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/filesystem"
	"github.com/sirupsen/logrus"
)

type CifsUploadHandler struct {
	cifs *filesystem.Cifs
}

func (u *CifsUploadHandler) WithCifs(cifs *filesystem.Cifs) *CifsUploadHandler {
	u.cifs = cifs

	return u
}

func (u *CifsUploadHandler) Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) (resErr error) {
	return forAllFilesInZip(file, func(f *zip.File) error {
		readCloser, err := f.Open()
		if err != nil {
			return err
		}
		defer readCloser.Close()

		logrus.Infof("Uploading file %s to CIFS share", f.Name)
		return u.cifs.UploadReader(f.Name, readCloser)
	})
}

func (u *CifsUploadHandler) Close() error {
	return nil
}

func extractTextFromPdf(pdfData []byte) (string, error) {
	return pdfToText(pdfData)
}

func pdfToText(pdfData []byte) (string, error) {
	cmd := exec.Command("pdftotext", "-", "-")
	inBuffer := bytes.NewBuffer(pdfData)
	outBuffer := &bytes.Buffer{}
	cmd.Stdin = inBuffer
	cmd.Stdout = outBuffer
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outBuffer.String(), nil
}
