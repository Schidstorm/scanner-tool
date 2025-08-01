package server

import (
	"archive/zip"
	"io"
	"os"

	"github.com/schidstorm/scanner-tool/pkg/ai"
	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/sirupsen/logrus"
)

type FileNameGuesserHandler struct {
	fileNameGuesser ai.FileNameGuesser
}

func NewFileNameGuesserHandler(fileNameGuesser ai.FileNameGuesser) *FileNameGuesserHandler {
	return &FileNameGuesserHandler{
		fileNameGuesser: fileNameGuesser,
	}
}

func (u *FileNameGuesserHandler) WithFileNameGuesser() *FileNameGuesserHandler {
	return u
}

func (i *FileNameGuesserHandler) Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) (resErr error) {
	resultZipFile, err := os.CreateTemp("", "scanner-tool-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(resultZipFile.Name())
	defer resultZipFile.Close()

	zipWriter := zip.NewWriter(resultZipFile)
	defer zipWriter.Close()

	pdfData, err := io.ReadAll(file)
	if err != nil {
		logrus.Errorf("Failed to read file: %v", err)
		return err
	}

	fileName, err := i.guessFileName(pdfData)
	if err != nil {
		logrus.Errorf("Failed to guess file name: %v", err)
	}

	pdfFileWithCorrectName, err := zipWriter.Create(fileName)
	if err != nil {
		return err
	}

	_, err = pdfFileWithCorrectName.Write(pdfData)
	if err != nil {
		logrus.Errorf("Failed to write PDF data to zip: %v", err)
	}

	defer zipWriter.Close()
	return outputQueue.EnqueueFilePath(resultZipFile.Name())
}

func (u *FileNameGuesserHandler) guessFileName(pdfData []byte) (string, error) {
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

func (i *FileNameGuesserHandler) Close() error {
	return nil
}
