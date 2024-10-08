package server

import (
	"archive/zip"
	"io"
	"os"
	"path"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/scan"
	"github.com/sirupsen/logrus"
)

type ScanHandler struct {
	scanner scan.Scanner
}

func (s *ScanHandler) WithScanner(scanner scan.Scanner) *ScanHandler {
	s.scanner = scanner

	return s
}

func (s *ScanHandler) Run(logger *logrus.Logger, inputQueue, outputQueue *filequeue.Queue) error {
	imagePaths, err := s.scanner.Scan()
	if err != nil {
		return err
	}

	if len(imagePaths) == 0 {
		return nil
	}

	logger.WithField("images", len(imagePaths)).Info("Scanned")

	tmpZip, err := makeTmpZipFile(imagePaths)
	if err != nil {
		return err
	}

	outputQueue.EnqueueFilePath(tmpZip)
	return nil
}

func makeTmpZipFile(contentFiles []string) (string, error) {
	tmpFile, err := os.CreateTemp("", "scanner-tool-*.zip")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	err = zipFiles(tmpFile, contentFiles)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func zipFiles(zipFile *os.File, contentFiles []string) error {
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range contentFiles {
		fileReader, err := os.Open(file)
		if err != nil {
			return err
		}
		defer fileReader.Close()

		fileContent, err := io.ReadAll(fileReader)
		if err != nil {
			return err
		}

		fileWriter, err := zipWriter.Create(path.Base(file))
		if err != nil {
			return err
		}

		_, err = fileWriter.Write(fileContent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ScanHandler) Close() error {
	return nil
}
