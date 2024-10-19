package scan

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

var productName = "DS-C490"

type Options struct {
	SaneOptions []string `yaml:"saneOptions"`
}

type SaneScanner struct {
	options      Options
	activeDevice string
}

func NewScanner(opts Options) *SaneScanner {
	s := &SaneScanner{
		options: opts,
	}

	return s
}

func (s *SaneScanner) Scan() ([]string, error) {
	if s.activeDevice == "" {
		device, err := detectDeviceByProductName(productName)
		if err != nil {
			return nil, err
		}

		s.activeDevice = device
	}

	scannedImages, err := s.execScanimage()
	if err != nil {
		return nil, err
	}

	return scannedImages, nil
}

func (s *SaneScanner) execScanimage() ([]string, error) {
	command := "scanimage"
	args := []string{"--format", "png", "--resolution", "600dpi", "--duplex=yes", "--batch=scan-%03d.png", "--batch-print", "--device-name", s.activeDevice}

	cmd := exec.Command(command, args...)
	imageFilesBuffer := &bytes.Buffer{}
	cmd.Stdout = imageFilesBuffer
	stdErrBuffer := &bytes.Buffer{}
	cmd.Stderr = stdErrBuffer
	err := cmd.Run()
	stderr := stdErrBuffer.String()
	if err != nil {
		if strings.Contains(stderr, "Error during device I/O") {
			// device is turned off
			return nil, nil
		}
		return nil, errors.Join(err, errors.New(stderr))
	}

	var result []string
	for _, imagePath := range strings.Split(imageFilesBuffer.String(), "\n") {
		if imagePath == "" {
			continue
		}

		result = append(result, strings.TrimSpace(imagePath))
	}

	return result, nil
}

// scanimage --format png --resolution 1200 --duplex=yes --batch --batch-print --device-name epsonscan2:DS-C490:584251413030303218:esci2:usb:ES0264:401

func loadImage(imagePath string) (image.Image, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func saveImage(imagePath string, img image.Image) error {
	file, err := os.Create(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func detectDeviceByProductName(productName string) (string, error) {
	logrus.WithField("productName", productName).Info("Detecting device by product name")
	devices, err := detectDevices()
	if err != nil {
		return "", err
	}

	for _, device := range devices {
		if strings.Contains(device, productName) {
			return device, nil
		}
	}

	return "", errors.New("device not found")
}

func detectDevices() ([]string, error) {
	logrus.Info("Detecting devices")
	// scanimage -L -d epson2
	deviceListOutput, err := execCommand("scanimage", "-L")
	if err != nil {
		logrus.WithError(err).Error("Failed to detect devices")
	}

	var devices []string
	for _, line := range strings.Split(deviceListOutput, "\n") {
		if strings.HasPrefix(line, "device") {
			deviceName := strings.Split(line, " ")[1]
			deviceName = removeAllTicks(deviceName)
			devices = append(devices, deviceName)
		}
	}

	return devices, nil
}

func removeAllTicks(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "'", ""), "`", "")
}

func execCommand(command string, args ...string) (string, error) {
	logrus.WithField("command", command).WithField("args", args).Info("Executing command")
	cmd := exec.Command(command, args...)
	outputBuffer := &bytes.Buffer{}
	cmd.Stdout = outputBuffer
	stderrBuffer := &bytes.Buffer{}
	cmd.Stderr = stderrBuffer
	err := cmd.Run()
	if err != nil {
		return "", errors.Join(err, errors.New(stderrBuffer.String()))
	}

	return outputBuffer.String(), nil
}
