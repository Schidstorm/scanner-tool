package server

import (
	"os"
	"testing"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/stretchr/testify/assert"
)

func prepareHandlerThings(t *testing.T, inoutFiles map[string][]byte, h func(input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) error) map[string][]byte {
	creator := queueoutputcreator.CreateZipFileWriter()

	for fileName, data := range inoutFiles {
		creator.AddFile(fileName, data)
	}

	zipFilePath, err := creator.Finalize()
	assert.NoError(t, err)
	defer os.Remove(zipFilePath)
	zipData, err := os.ReadFile(zipFilePath)
	assert.NoError(t, err)

	inputZipFile := &filequeue.MemQueryFile{
		Name: "test.zip",
		Data: zipData,
	}

	inputFiles := make(chan InputFile)
	go func() {
		defer close(inputFiles)
		zipReader, err := queueoutputcreator.CreateZipFileReader(inputZipFile)
		assert.NoError(t, err)
		for _, fileName := range zipReader.FileNames() {
			file, err := zipReader.GetFile(fileName)
			assert.NoError(t, err)
			inputFiles <- file
		}
	}()

	outputZipCreator := queueoutputcreator.CreateMemZipFileCreator()

	err = h(inputFiles, outputZipCreator)
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(outputZipCreator.Files()))

	result := make(map[string][]byte)
	for fileName, buf := range outputZipCreator.Files() {
		result[fileName] = buf.Bytes()
	}
	return result
}
