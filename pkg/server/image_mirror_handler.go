package server

import (
	"image"
	"image/png"

	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/sirupsen/logrus"
)

type ImageMirrorHandler struct {
}

func (i *ImageMirrorHandler) Run(logger *logrus.Logger, input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) (resErr error) {
	for f := range input {
		img, err := readImage(f)
		if err != nil {
			logger.Errorf("Failed to read image from file %s: %v", f.FileInfo().Name(), err)
			return err
		}

		resultImage := mirrorImage(img)
		file := outputFiles.OpenFile(f.FileInfo().Name())
		err = png.Encode(file, resultImage)
		if err != nil {
			return err
		}
	}

	return nil
}

func readImage(f InputFile) (image.Image, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	img, err := png.Decode(rc)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func mirrorImage(img image.Image) image.Image {
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
