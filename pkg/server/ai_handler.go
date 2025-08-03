package server

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	"github.com/schidstorm/scanner-tool/pkg/ai"
	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/sirupsen/logrus"
)

type AiHandler struct {
	fileNameGuesser ai.FileNameGuesser
	fileTagsGuesser ai.FileTagsGuesser
}

func (i *AiHandler) WithFileNameGuesser(fileNameGuesser ai.FileNameGuesser) *AiHandler {
	i.fileNameGuesser = fileNameGuesser
	return i
}

func (i *AiHandler) WithFileTagsGuesser(fileTagsGuesser ai.FileTagsGuesser) *AiHandler {
	i.fileTagsGuesser = fileTagsGuesser
	return i
}

func (i *AiHandler) Run(logger *logrus.Logger, input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) (resErr error) {
	for f := range input {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		pdfData, err := io.ReadAll(rc)
		if err != nil {
			return err
		}

		fileName, fileTags, err := i.guessFileNameAndTags(pdfData)
		if err != nil {
			logrus.Errorf("Failed to guess file name: %v", err)
			return err
		}

		outputFiles.AddFile(fileName, pdfData)
		metadata := &queueoutputcreator.Metadata{}
		metadata.Set("tags", strings.Join(fileTags, ","))
		outputFiles.AttachMetadata(fileName, metadata)
	}

	return nil
}

func (i *AiHandler) guessFileNameAndTags(pdfData []byte) (string, []string, error) {
	logrus.Info("Extracting text from PDF")
	text, err := extractTextFromPdf(pdfData)
	if err != nil {
		logrus.Errorf("Failed to extract text from PDF: %v", err)
		return "", nil, err
	}
	logrus.Info("Guessing file name")
	fileName, err := i.fileNameGuesser.Guess(text)
	if err != nil {
		logrus.Errorf("Failed to guess file name: %v", err)
		return "", nil, err
	}
	logrus.Infof("Guessed file name: %s", fileName)

	logrus.Info("Guessing file tags")
	fileTags, err := i.fileTagsGuesser.Guess(text)
	if err != nil {
		logrus.Errorf("Failed to guess file tags: %v", err)
		return "", nil, err
	}
	logrus.Infof("Guessed file tags: %v", fileTags)

	return fileName, fileTags, nil
}

func extractTextFromPdf(pdfData []byte) (string, error) {
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

func (i *AiHandler) Close() error {
	return nil
}
