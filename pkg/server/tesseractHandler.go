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
		pdfFile, err := zipWriter.Create(pdfFileName(f.Name))
		if err != nil {
			return err
		}

		fileHandle, err := f.Open()
		if err != nil {
			return err
		}
		defer fileHandle.Close()

		err = tesseract.ConvertImageToPdf(fileHandle, pdfFile)
		if err != nil {
			logger.WithError(err).Error("Failed to convert image to pdf")
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	zipWriter.Close()
	return outputQueue.EnqueueFilePath(resultZipFile.Name())
}

func pdfFileName(fileName string) string {
	return strings.TrimSuffix(fileName, ".png") + ".pdf"
}

func (t *TesseractHandler) Close() error {
	return nil
}
