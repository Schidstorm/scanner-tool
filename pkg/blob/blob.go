package blob

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/schidstorm/scanner-tool/pkg/filesystem"
)

type Metadata struct {
	Pdf   string
	Image string
}

type Blob struct {
	cifs        *filesystem.Cifs
	archiveFile string
	metadata    *Metadata
}

func LoadBlob(cifs *filesystem.Cifs, archiveFile string) *Blob {
	return &Blob{
		cifs:        cifs,
		archiveFile: archiveFile,
	}
}

func (b *Blob) Metadata() (*Metadata, error) {
	metaFile, err := b.ReadFile("metadata.json")
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{}
	err = json.Unmarshal(metaFile, metadata)
	if err != nil {
		return nil, err
	}

	b.metadata = metadata

	return metadata, nil
}

func (b *Blob) ReadFile(name string) ([]byte, error) {
	fileContent, err := b.cifs.Download(b.archiveFile)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(fileContent), int64(len(fileContent)))
	if err != nil {
		return nil, err
	}

	for _, file := range zipReader.File {
		if file.Name == name {
			fileReader, err := file.Open()
			if err != nil {
				return nil, err
			}

			return io.ReadAll(fileReader)
		}
	}

	return nil, os.ErrNotExist
}

func (b *Blob) WriteFile(name string, content []byte) error {
	fileContent, err := b.cifs.Download(b.archiveFile)
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(fileContent), int64(len(fileContent)))
	if err != nil {
		return err
	}

	var newZipContent bytes.Buffer
	zipWriter := zip.NewWriter(&newZipContent)
	for _, file := range zipReader.File {
		if file.Name != name {
			fileReader, err := file.Open()
			if err != nil {
				return err
			}

			fileContent, err := io.ReadAll(fileReader)
			if err != nil {
				return err
			}

			fileWriter, err := zipWriter.Create(file.Name)
			if err != nil {
				return err
			}

			_, err = fileWriter.Write(fileContent)
			if err != nil {
				return err
			}
		} else {
			fileWriter, err := zipWriter.Create(name)
			if err != nil {
				return err
			}

			_, err = fileWriter.Write(content)
			if err != nil {
				return err
			}
		}
	}

	zipWriter.Close()

	return b.cifs.Upload(b.archiveFile, newZipContent.Bytes())
}
