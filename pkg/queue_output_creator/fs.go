package queueoutputcreator

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
)

type FsZipFileWriter struct {
	file      *os.File
	zipWriter *zip.Writer
	err       error
	fileCount int
}

func CreateZipFileWriter() QueueZipFileWriter {
	result := &FsZipFileWriter{}

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

func (z FsZipFileWriter) FileCount() int {
	if z.err != nil {
		return 0
	}
	return z.fileCount
}

func (z *FsZipFileWriter) Error() error {
	return z.err
}

func (z *FsZipFileWriter) OpenFile(fileName string) io.Writer {
	if z.err != nil {
		return &nullWriter{}
	}

	file, err := z.zipWriter.Create(fileName)
	if err != nil {
		z.err = err
		return &nullWriter{}
	}

	z.fileCount++
	return file
}

func (z *FsZipFileWriter) AddFile(fileName string, data []byte) QueueZipFileWriter {
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

	z.fileCount++

	return z
}

func (z *FsZipFileWriter) AddFileReader(fileName string, r io.Reader) QueueZipFileWriter {
	b := bytes.NewBuffer(nil)
	io.Copy(b, r)
	return z.AddFile(fileName, b.Bytes())
}

func (z *FsZipFileWriter) Finalize() (string, error) {
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

func (z *FsZipFileWriter) AttachMetadata(fileName string, metadata *Metadata) QueueZipFileWriter {
	if z.err != nil {
		return z
	}

	if metadata == nil || metadata.IsEmpty() {
		return z
	}

	return z.AddFile(".metadata."+fileName, metadata.Serialize())
}

type FsZipFileReader struct {
	zipReader *zip.Reader
	files     map[string]*zip.File
	metadata  map[string]*Metadata
}

func CreateZipFileReader(queueFile filequeue.QueueFile) (QueueZipFileReader, error) {
	fileSize, err := queueFile.Size()
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(queueFile, fileSize)
	if err != nil {
		return nil, err
	}

	files := make(map[string]*zip.File)
	metadata := make(map[string]*Metadata)
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if strings.HasPrefix(file.Name, ".metadata.") {
			zfContent, err := readZipFile(file)
			if err != nil {
				return nil, err
			}
			metadata[file.Name] = DeserializeMetadata(zfContent)
		} else {
			files[file.Name] = file
		}
	}

	return &FsZipFileReader{
		zipReader: zipReader,
		files:     files,
		metadata:  metadata,
	}, nil
}

func readZipFile(zf *zip.File) ([]byte, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (z *FsZipFileReader) GetFile(fileName string) (*ZipFile, error) {
	metadata := z.metadata[".metadata."+fileName]
	if metadata == nil {
		metadata = &Metadata{}
	}

	if file, exists := z.files[fileName]; exists {
		return &ZipFile{
			File:     file,
			metadata: z.metadata[".metadata."+fileName],
		}, nil
	}

	return nil, os.ErrNotExist
}

func (z *FsZipFileReader) FileNames() []string {

	fileNames := make([]string, 0, len(z.files))
	for fileName := range z.files {
		fileNames = append(fileNames, fileName)
	}

	return fileNames
}
