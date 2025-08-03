package server

import (
	"os"

	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
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

func (s *ScanHandler) Run(logger *logrus.Logger, _ chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) error {
	imagePaths, err := s.scanner.Scan()
	if err != nil {
		return err
	}

	if len(imagePaths) == 0 {
		return nil
	}

	logger.WithField("images", len(imagePaths)).WithField("files", imagePaths).Info("Scanned")

	for _, imagePath := range imagePaths {
		rc, err := os.OpenFile(imagePath, os.O_RDONLY, 0o644)
		if err != nil {
			return err
		}
		defer rc.Close()

		fileInfo, err := rc.Stat()
		if err != nil {
			return err
		}

		fileName := fileInfo.Name()
		outputFiles.AddFileReader(fileName, rc)
	}

	return nil
}

func (s *ScanHandler) Close() error {
	return nil
}
