package server

import (
	"strings"

	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/schidstorm/scanner-tool/pkg/tesseract"
	"github.com/sirupsen/logrus"
)

type TesseractHandler struct {
}

func (t *TesseractHandler) Run(logger *logrus.Logger, input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) (resErr error) {
	for f := range input {
		text, err := fileToText(f)
		if err != nil {
			return err
		}

		if strings.Trim(text, " \n") == "" {
			continue
		}

		err = fileToPdf(f, outputFiles)
		if err != nil {
			return err
		}
	}

	return nil
}

func fileToText(zipFile InputFile) (string, error) {
	fileHandle, err := zipFile.Open()
	if err != nil {
		return "", err
	}
	defer fileHandle.Close()

	return tesseract.ConvertImageToText(fileHandle)
}

func fileToPdf(zipFile InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) error {
	fileHandle, err := zipFile.Open()
	if err != nil {
		return err
	}
	defer fileHandle.Close()

	pdfWriter := outputFiles.OpenFile(pdfFileName(zipFile.FileInfo().Name()))

	return tesseract.ConvertImageToPdf(fileHandle, pdfWriter)
}

func pdfFileName(fileName string) string {
	return strings.TrimSuffix(fileName, ".png") + ".pdf"
}

func (t *TesseractHandler) Close() error {
	return nil
}
