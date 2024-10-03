package scan

import (
	"errors"
	"io"
	"math/rand/v2"
	"os"
	"path"
)

var ErrNoNewDocuments = errors.New("no new documents")

type Scanner interface {
	Scan(outputImage io.Writer) error
}

var fakeImagePoolFolder = "fakeImages"

type FakeScanner struct {
}

func NewFakeScanner() *FakeScanner {
	return &FakeScanner{}
}

func (s *FakeScanner) Scan(outputImage io.Writer) error {
	returnEarly := rand.IntN(3) == 0
	if returnEarly {
		return ErrNoNewDocuments
	}

	// list images
	files, err := os.ReadDir(fakeImagePoolFolder)
	if err != nil {
		return err
	}

	// select one
	selectedFile := files[rand.IntN(len(files))]

	// read image
	selectedImage, err := os.ReadFile(path.Join(fakeImagePoolFolder, selectedFile.Name()))
	if err != nil {
		return err
	}

	_, err = outputImage.Write(selectedImage)
	return err
}
