package server

import (
	"archive/zip"
	"bytes"
	"io"
	"os"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
)

type zipFileCreator struct {
	file      *os.File
	zipWriter *zip.Writer
	err       error
}

type nullWriter struct{}

func (n *nullWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func createZipFileCreator() *zipFileCreator {
	result := &zipFileCreator{}

	resultZipFile, err := os.CreateTemp("", "scanner-tool-*.zip")
	if err != nil {
		result.err = err
		return result
	}
	result.file = resultZipFile

	zipWriter := zip.NewWriter(resultZipFile)
	result.zipWriter = zipWriter

	return result
}

func (z *zipFileCreator) OpenFile(fileName string) io.Writer {
	if z.err != nil {
		return &nullWriter{}
	}

	file, err := z.zipWriter.Create(fileName)
	if err != nil {
		z.err = err
		return &nullWriter{}
	}

	return file
}

func (z *zipFileCreator) AddFile(fileName string, data []byte) *zipFileCreator {
	if z.err != nil {
		return z
	}

	file, err := z.zipWriter.Create(fileName)
	if err != nil {
		z.err = err
		return z
	}

	_, err = file.Write(data)
	if err != nil {
		z.err = err
	}

	return z
}

func (z *zipFileCreator) AddFileReader(fileName string, r io.Reader) *zipFileCreator {
	b := bytes.NewBuffer(nil)
	io.Copy(b, r)
	return z.AddFile(fileName, b.Bytes())
}

func (z *zipFileCreator) Finalize() (string, error) {
	if z.err != nil {
		return "", z.err
	}

	if z.zipWriter != nil {
		z.zipWriter.Close()
	}

	filePath := z.file.Name()
	if z.file != nil {
		z.file.Close()
	}

	return filePath, nil
}

func forAllFilesInZip(zipFile filequeue.QueueFile, handler func(f *zip.File) error) error {
	fileSize, err := zipFile.Size()
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(zipFile, fileSize)
	if err != nil {
		return err
	}

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		err = handler(file)
		fileReader.Close()
		if err != nil {
			return err
		}

	}

	return nil
}
