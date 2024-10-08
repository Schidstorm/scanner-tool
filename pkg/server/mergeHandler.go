package server

import (
	"archive/zip"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/sirupsen/logrus"
)

type MergeHandler struct {
}

func (m *MergeHandler) Run(logger *logrus.Logger, inputQueue, outputQueue *filequeue.Queue) error {
	file, err := inputQueue.Dequeue()
	if err != nil {
		return err
	}
	if file == nil {
		return nil
	}
	defer file.Close()

	var tmpFiles []string
	defer func() {
		for _, tmpFile := range tmpFiles {
			os.Remove(tmpFile)
		}
	}()

	err = forAllFilesInZip(file.File, func(f *zip.File) error {
		fileHandle, err := f.Open()
		if err != nil {
			return err
		}
		defer fileHandle.Close()

		tmpFile, err := os.CreateTemp("", "scanner-tool-*.pdf")
		if err != nil {
			return err
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, fileHandle)
		if err != nil {
			return err
		}

		tmpFiles = append(tmpFiles, tmpFile.Name())
		return nil
	})
	if err != nil {
		return err
	}

	tmpMergedFile, err := os.CreateTemp("", "scanner-tool-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmpMergedFile.Name())
	defer tmpMergedFile.Close()

	readers := make([]io.ReadSeeker, 0, len(tmpFiles))
	defer func() {
		for _, tmpFile := range readers {
			if r, ok := tmpFile.(io.Closer); ok {
				r.Close()
			}
		}
	}()

	for _, tmpFile := range tmpFiles {
		file, err := os.Open(tmpFile)
		if err != nil {
			return err
		}
		readers = append(readers, file)
	}

	err = api.MergeRaw(readers, tmpMergedFile, false, model.NewDefaultConfiguration())
	if err != nil {
		return err
	}

	return outputQueue.EnqueueFilePath(tmpMergedFile.Name())
}

func (m *MergeHandler) Close() error {
	return nil
}
