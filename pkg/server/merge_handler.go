package server

import (
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/sirupsen/logrus"
)

type MergeHandler struct {
}

func (m *MergeHandler) Run(logger *logrus.Logger, input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) (resErr error) {
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

	tmpFiles, err = unpackAllFilesInZip(input, tmpDir)
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

	outputFiles.AddFile("out.pdf", zipBytes)

	return nil
}

func unpackAllFilesInZip(files chan InputFile, destDir string) ([]string, error) {
	var tmpFiles []string
	var loopErr error
	for f := range files {
		fileHandle, err := f.Open()
		if err != nil {
			loopErr = err
			break
		}
		defer fileHandle.Close()

		tmpFilePath := path.Join(destDir, f.FileInfo().Name())
		tmpFile, err := os.Create(tmpFilePath)
		if err != nil {
			loopErr = err
			break
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, fileHandle)
		if err != nil {
			loopErr = err
			break
		}

		tmpFiles = append(tmpFiles, tmpFilePath)
	}

	if loopErr != nil {
		for _, tmpFile := range tmpFiles {
			os.Remove(tmpFile)
		}
		return nil, loopErr
	}

	sort.Strings(tmpFiles)

	return tmpFiles, nil
}

func (m *MergeHandler) Close() error {
	return nil
}
