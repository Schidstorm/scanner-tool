package server

import (
	"archive/zip"
	"io"
	"os"
	"testing"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/stretchr/testify/assert"
)

func prepareHandlerThings(t *testing.T, inoutFiles map[string][]byte, h func(file filequeue.QueueFile, outputQueue filequeue.Queue) error) map[string][]byte {
	creator := createZipFileCreator()

	for fileName, data := range inoutFiles {
		creator.AddFile(fileName, data)
	}

	zipFilePath, err := creator.Finalize()
	assert.NoError(t, err)
	defer os.Remove(zipFilePath)
	zipData, err := os.ReadFile(zipFilePath)
	assert.NoError(t, err)

	file := &filequeue.MemQueryFile{
		Name: "test.zip",
		Data: zipData,
	}

	outputQueue := &filequeue.MemQueryFileQueue{}
	err = h(file, outputQueue)
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(outputQueue.Files))

	result := make(map[string][]byte)
	forAllFilesInZip(&outputQueue.Files[0], func(f *zip.File) error {
		readCloser, err := f.Open()
		assert.NoError(t, err)
		defer readCloser.Close()
		imgData, err := io.ReadAll(readCloser)
		assert.NoError(t, err)
		result[f.Name] = imgData
		return nil
	})
	return result
}
