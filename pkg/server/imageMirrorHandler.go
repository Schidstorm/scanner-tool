package server

import (
	"archive/zip"
	"image"
	"image/png"
	"os"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/sirupsen/logrus"
)

type ImageMirrorHandler struct {
}

func (i *ImageMirrorHandler) Run(logger *logrus.Logger, inputQueue, outputQueue *filequeue.Queue) error {
	file, err := inputQueue.Dequeue()
	if err != nil {
		return err
	}
	if file == nil {
		return nil
	}
	defer file.Close()

	resultZipFile, err := os.CreateTemp("", "scanner-tool-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(resultZipFile.Name())
	defer resultZipFile.Close()

	zipWriter := zip.NewWriter(resultZipFile)
	defer zipWriter.Close()

	err = forAllFilesInZip(file.File, func(f *zip.File) error {
		resultImage := mirrorImage(f)
		if resultImage == nil {
			return nil
		}

		file, err := zipWriter.Create(f.Name)
		if err != nil {
			return err
		}

		err = png.Encode(file, resultImage)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	defer zipWriter.Close()
	return outputQueue.EnqueueFilePath(resultZipFile.Name())
}

func forAllFilesInZip(zipFile *os.File, handler func(f *zip.File) error) error {
	stat, err := zipFile.Stat()
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(zipFile, stat.Size())
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

func mirrorImage(f *zip.File) image.Image {
	fileHandle, err := f.Open()
	if err != nil {
		return nil
	}

	img, err := png.Decode(fileHandle)
	if err != nil {
		return nil
	}

	resultImage := image.NewRGBA(img.Bounds())
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			resultImage.Set(x, y, img.At(img.Bounds().Dx()-x-1, img.Bounds().Dy()-y-1))
		}
	}
	return resultImage
}

func (i *ImageMirrorHandler) Close() error {
	return nil
}
