package server

import (
	"archive/zip"
	"os"
	"strings"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/tesseract"
	"github.com/sirupsen/logrus"
)

type TesseractHandler struct {
}

func (t *TesseractHandler) Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) (resErr error) {
	resultZipFile, err := os.CreateTemp("", "scanner-tool-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(resultZipFile.Name())
	defer resultZipFile.Close()

	zipWriter := zip.NewWriter(resultZipFile)
	defer zipWriter.Close()
	err = forAllFilesInZip(file, func(f *zip.File) error {
		text, err := fileToText(f)
		if err != nil {
			return err
		}

		if strings.Trim(text, " \n") == "" {
			logger.Info("No text found in image. Skipping pdf conversion")
			return nil
		}

		return fileToPdf(f, zipWriter)
	})
	if err != nil {
		return err
	}

	zipWriter.Close()
	return outputQueue.EnqueueFilePath(resultZipFile.Name())
}

func fileToText(zipFile *zip.File) (string, error) {
	fileHandle, err := zipFile.Open()
	if err != nil {
		return "", err
	}
	defer fileHandle.Close()

	return tesseract.ConvertImageToText(fileHandle)
}

func fileToPdf(zipFile *zip.File, zipWriter *zip.Writer) error {
	fileHandle, err := zipFile.Open()
	if err != nil {
		return err
	}
	defer fileHandle.Close()

	pdfWriter, err := zipWriter.Create(pdfFileName(zipFile.Name))
	if err != nil {
		return err
	}

	return tesseract.ConvertImageToPdf(fileHandle, pdfWriter)
}

func pdfFileName(fileName string) string {
	return strings.TrimSuffix(fileName, ".png") + ".pdf"
}

func (t *TesseractHandler) Close() error {
	return nil
}
