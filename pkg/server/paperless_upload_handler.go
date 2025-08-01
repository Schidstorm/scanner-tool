package server

import (
	"bytes"
	"io"

	"github.com/schidstorm/scanner-tool/pkg/ai"
	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/paperless"
	"github.com/sirupsen/logrus"
)

type PaperlessUploadHandler struct {
	fileNameGuesser ai.FileNameGuesser
	fileTagsGuesser ai.FileTagsGuesser
	paperless       *paperless.Paperless
}

func (u *PaperlessUploadHandler) WithPaperless(paperless *paperless.Paperless) *PaperlessUploadHandler {
	u.paperless = paperless
	return u
}

func (u *PaperlessUploadHandler) WithFileNameGuesser(fileNameGuesser ai.FileNameGuesser) *PaperlessUploadHandler {
	u.fileNameGuesser = fileNameGuesser
	return u
}

func (u *PaperlessUploadHandler) WithFileTagsGuesser(fileTagsGuesser ai.FileTagsGuesser) *PaperlessUploadHandler {
	u.fileTagsGuesser = fileTagsGuesser
	return u
}

func (u *PaperlessUploadHandler) Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) (resErr error) {
	pdfData, err := io.ReadAll(file)
	if err != nil {
		logrus.Errorf("Failed to read file: %v", err)
		return err
	}

	fileName, fileTags, err := u.guessFileNameAndTags(pdfData)
	if err != nil {
		logrus.Errorf("Failed to guess file name: %v", err)
		return err
	}
	logrus.Infof("Uploading file %s to Paperless", fileName)

	fileReader := bytes.NewReader(pdfData)
	return u.paperless.Upload(fileReader, paperless.UploadOptions{
		Title: fileName,
		Tags:  fileTags,
	})
}

func (u *PaperlessUploadHandler) guessFileNameAndTags(pdfData []byte) (string, []string, error) {
	logrus.Info("Extracting text from PDF")
	text, err := extractTextFromPdf(pdfData)
	if err != nil {
		logrus.Errorf("Failed to extract text from PDF: %v", err)
		return "", nil, err
	}
	logrus.Info("Guessing file name")
	fileName, err := u.fileNameGuesser.Guess(text)
	if err != nil {
		logrus.Errorf("Failed to guess file name: %v", err)
		return "", nil, err
	}
	logrus.Infof("Guessed file name: %s", fileName)

	logrus.Info("Guessing file tags")
	fileTags, err := u.fileTagsGuesser.Guess(text)
	if err != nil {
		logrus.Errorf("Failed to guess file tags: %v", err)
		return "", nil, err
	}
	logrus.Infof("Guessed file tags: %v", fileTags)

	return fileName, fileTags, nil
}

func (u *PaperlessUploadHandler) Close() error {
	return nil
}
