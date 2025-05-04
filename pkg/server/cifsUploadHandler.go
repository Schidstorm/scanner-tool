package server

import (
	"bytes"
	"io"
	"os/exec"

	"github.com/schidstorm/scanner-tool/pkg/ai"
	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/filesystem"
	"github.com/sirupsen/logrus"
)

type CifsUploadHandler struct {
	cifs            *filesystem.Cifs
	fileNameGuesser ai.FileNameGuesser
}

func (u *CifsUploadHandler) WithCifs(cifs *filesystem.Cifs) *CifsUploadHandler {
	u.cifs = cifs

	return u
}

func (u *CifsUploadHandler) WithFileNameGuesser(fileNameGuesser ai.FileNameGuesser) *CifsUploadHandler {
	u.fileNameGuesser = fileNameGuesser
	return u
}

func (u *CifsUploadHandler) Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) (resErr error) {
	pdfData, err := io.ReadAll(file)
	if err != nil {
		logrus.Errorf("Failed to read file: %v", err)
		return err
	}

	fileName, err := u.guessFileName(pdfData)
	if err != nil {
		logrus.Errorf("Failed to guess file name: %v", err)
		return err
	}
	logrus.Infof("Uploading file %s to CIFS share", fileName)
	return u.cifs.Upload(fileName, pdfData)
}

func (u *CifsUploadHandler) guessFileName(pdfData []byte) (string, error) {
	logrus.Info("Extracting text from PDF")
	text, err := extractTextFromPdf(pdfData)
	if err != nil {
		logrus.Errorf("Failed to extract text from PDF: %v", err)
		return "", err
	}
	logrus.Info("Guessing file name")
	fileName, err := u.fileNameGuesser.Guess(text)
	if err != nil {
		logrus.Errorf("Failed to guess file name: %v", err)
		return "", err
	}
	logrus.Infof("Guessed file name: %s", fileName)

	return fileName, nil
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
