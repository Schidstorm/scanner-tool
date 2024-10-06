package server

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/schidstorm/scanner-tool/pkg/filesystem"
	"github.com/schidstorm/scanner-tool/pkg/scan"
	"github.com/schidstorm/scanner-tool/pkg/tesseract"
	"github.com/sirupsen/logrus"
)

var daemonScanWait = 10 * time.Second
var filePrefix = "zzzzzzz-scan-"
var todoFolder = path.Join(os.TempDir(), "sscanner-tool-todo")

type DaemonHandler interface {
	Run() error
	Close() error
}

type Daemon struct {
	closeRequest chan struct{}
	handlers     []DaemonHandler
	wgClosed     *sync.WaitGroup
}

func NewDaemon(handlers []DaemonHandler) *Daemon {
	return &Daemon{
		handlers: handlers,
		wgClosed: new(sync.WaitGroup),
	}
}

func (d *Daemon) Start() error {
	d.closeRequest = make(chan struct{})
	d.wgClosed.Add(len(d.handlers))

	for _, handler := range d.handlers {
		go d.run(handler)
	}
	return nil
}

func (d *Daemon) Stop() error {
	close(d.closeRequest)
	d.wgClosed.Wait()
	return nil
}

func (d *Daemon) run(handler DaemonHandler) {
	for {
		select {
		case <-d.closeRequest:
			handler.Close()
			d.wgClosed.Done()
			return
		case <-time.After(daemonScanWait):
		}

		d.runHandler(handler)
	}
}

func (d *Daemon) runHandler(handler DaemonHandler) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("recover", r).Error("Recovered")
		}
	}()

	err := handler.Run()
	if err != nil {
		logrus.WithError(err).Error("Failed to run handler")
	}
}

type ScanHandler struct {
	scanner scan.Scanner
}

func (s *ScanHandler) WithScanner(scanner scan.Scanner) *ScanHandler {
	s.scanner = scanner

	return s
}

func (s *ScanHandler) Run() error {
	logrus.Info("Scanning")
	imagePaths, err := s.scanner.Scan()
	if err != nil {
		return err
	}
	logrus.WithField("images", len(imagePaths)).Info("Scanned")

	imageHashes, err := calcFileHashes(imagePaths)
	if err != nil {
		return err
	}
	logrus.WithField("images", len(imageHashes)).Info("Hashed")

	err = os.MkdirAll(todoFolder, 0755)
	if err != nil {
		return err
	}

	for _, imagePath := range imagePaths {
		hash := imageHashes[imagePath]
		todoFileName := fmt.Sprintf("%d_%d_%s_%s", time.Now().Unix(), time.Now().Nanosecond(), path.Base(imagePath), hash)
		todoPath := path.Join(todoFolder, todoFileName)
		err = os.Rename(imagePath, todoPath)
		if err != nil {
			logrus.WithError(err).Error("Failed to move image to todo folder")
			continue
		}
	}
	logrus.Info("Moved images to todo folder")

	return nil
}

func calcFileHashes(files []string) (map[string]string, error) {
	hashes := make(map[string]string)
	for _, file := range files {
		hash, err := calcFileHash(file)
		if err != nil {
			return nil, err
		}

		hashes[file] = hash
	}

	return hashes, nil
}

func calcFileHash(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

func (s *ScanHandler) Close() error {
	return nil
}

type TesseractHandler struct {
	cifs *filesystem.Cifs
}

func (t *TesseractHandler) WithCifs(cifs *filesystem.Cifs) *TesseractHandler {
	t.cifs = cifs

	return t
}

func (t *TesseractHandler) Run() error {
	logrus.Info("Running tesseract")
	files, err := os.ReadDir(todoFolder)
	if err != nil {
		return err
	}
	logrus.WithField("files", len(files)).Info("Found files")

	if len(files) == 0 {
		return nil
	}

	firstFile := files[0]
	f, err := os.Open(path.Join(todoFolder, firstFile.Name()))
	if err != nil {
		return err
	}
	defer f.Close()

	pdfBuffer := new(bytes.Buffer)
	err = tesseract.ConvertImageToPdf(f, pdfBuffer)
	if err != nil {
		logrus.WithError(err).Error("Failed to convert image to pdf")
		return err
	}

	fileId, err := t.cifs.NextFileId()
	if err != nil {
		logrus.WithError(err).Error("Failed to get next file id")
		return err
	}

	pdfPath := fmt.Sprintf("%s%02d.pdf", filePrefix, fileId)
	pngPath := fmt.Sprintf("%s%02d.png", filePrefix, fileId)

	err = t.cifs.Upload(pdfPath, pdfBuffer.Bytes())
	if err != nil {
		logrus.WithError(err).Error("Failed to upload pdf")
		return err
	}
	pdfBuffer.Reset()

	f.Seek(0, 0)
	imgData, err := io.ReadAll(f)
	if err != nil {
		logrus.WithError(err).Error("Failed to read image data")
		return err
	}

	err = t.cifs.Upload(pngPath, imgData)
	if err != nil {
		logrus.WithError(err).Error("Failed to upload image")
		return err
	}

	err = os.Remove(path.Join(todoFolder, firstFile.Name()))
	if err != nil {
		logrus.WithError(err).Error("Failed to remove file")
		return err
	}

	return nil
}

func (t *TesseractHandler) Close() error {
	return nil
}
