package tesseract

import (
	"bytes"
	"errors"
	"io"
	"os/exec"

	"github.com/sirupsen/logrus"
)

func ConvertImageToPdf(inputImage io.Reader, output io.Writer) error {
	logrus.Info("Converting image to pdf")
	cmd := exec.Command("tesseract", "-", "-", "pdf")
	cmd.Stdin = inputImage
	cmd.Stdout = output
	errorBuffer := &bytes.Buffer{}
	cmd.Stderr = errorBuffer
	err := cmd.Run()

	if err != nil {
		return errors.Join(err, errors.New(errorBuffer.String()))
	}

	return nil
}

func ConvertImageToText(inputImage io.Reader) (string, error) {
	logrus.Info("Converting image to text")
	cmd := exec.Command("tesseract", "-", "-")
	cmd.Stdin = inputImage
	outputBuffer := &bytes.Buffer{}
	cmd.Stdout = outputBuffer
	errorBuffer := &bytes.Buffer{}
	cmd.Stderr = errorBuffer
	err := cmd.Run()

	if err != nil {
		return "", errors.Join(err, errors.New(errorBuffer.String()))
	}

	return outputBuffer.String(), nil
}
