package server

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/sirupsen/logrus"
)

type MergeHandler struct {
}

func (m *MergeHandler) Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) (resErr error) {
	var tmpFiles []string
	defer func() {
		for _, tmpFile := range tmpFiles {
			os.Remove(tmpFile)
		}
	}()

	tmpDir, err := os.MkdirTemp("", "scanner-tool-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tmpFiles, err = unpackAllFilesInZip(file, tmpDir)
	if err != nil {
		return err
	}

	tmpMergedFileName := fmt.Sprintf("%d_%d.pdf", time.Now().Unix(), time.Now().Nanosecond())
	tmpMergedFilePath := path.Join(os.TempDir(), tmpMergedFileName)

	err = api.MergeCreateFile(tmpFiles, tmpMergedFilePath, false, model.NewDefaultConfiguration())
	if err != nil {
		return err
	}

	zipBytes, err := os.ReadFile(tmpMergedFilePath)
	if err != nil {
		return err
	}

	ouputZipPath, err := createZipFileCreator().
		AddFile("out.pdf", zipBytes).
		Finalize()
	if err != nil {
		return err
	}

	return outputQueue.EnqueueFilePath(ouputZipPath)
}

func unpackAllFilesInZip(zipFile filequeue.QueueFile, destDir string) ([]string, error) {
	var tmpFiles []string
	err := forAllFilesInZip(zipFile, func(f *zip.File) error {
		fileHandle, err := f.Open()
		if err != nil {
			return err
		}
		defer fileHandle.Close()

		tmpFilePath := path.Join(destDir, f.Name)
		tmpFile, err := os.Create(tmpFilePath)
		if err != nil {
			return err
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, fileHandle)
		if err != nil {
			return err
		}

		tmpFiles = append(tmpFiles, tmpFilePath)
		return nil
	})

	if err != nil {
		for _, tmpFile := range tmpFiles {
			os.Remove(tmpFile)
		}
		return nil, err
	}

	sort.Strings(tmpFiles)

	return tmpFiles, nil
}

func (m *MergeHandler) Close() error {
	return nil
}
