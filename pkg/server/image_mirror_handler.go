package server

import (
	"archive/zip"
	"image"
	"image/png"

	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/sirupsen/logrus"
)

type ImageMirrorHandler struct {
}

func (i *ImageMirrorHandler) Run(logger *logrus.Logger, file filequeue.QueueFile, outputQueue filequeue.Queue) (resErr error) {
	zipFile := createZipFileCreator()

	err := forAllFilesInZip(file, func(f *zip.File) error {
		resultImage := mirrorImage(f)
		if resultImage == nil {
			return nil
		}

		file := zipFile.OpenFile(f.Name)
		err := png.Encode(file, resultImage)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	zipPath, err := zipFile.Finalize()
	if err != nil {
		return err
	}
	return outputQueue.EnqueueFilePath(zipPath)
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
