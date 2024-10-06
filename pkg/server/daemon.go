package server

import (
	"archive/zip"
	"image"
	"image/png"
	"io"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/filesystem"
	"github.com/schidstorm/scanner-tool/pkg/scan"
	"github.com/schidstorm/scanner-tool/pkg/tesseract"
	"github.com/sirupsen/logrus"
)

var daemonScanWait = 5 * time.Second

var queueScans = filequeue.NewQueue("new-scans")
var queueRotated = filequeue.NewQueue("new-rotated")
var queueTesseracted = filequeue.NewQueue("new-tesseracted")
var queueMerged = filequeue.NewQueue("new-merged")

type DaemonHandler interface {
	Run(logger *logrus.Logger) error
	Close() error
}

type Daemon struct {
	closeRequest chan struct{}
	handlers     []DaemonHandler
	wgClosed     *sync.WaitGroup
}

type daemonLogger struct {
	daemonName string
	formatter  logrus.Formatter
}

func (d daemonLogger) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Data["daemon"] = d.daemonName
	return d.formatter.Format(entry)
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
	typeName := reflect.TypeOf(handler).String()

	logger := logrus.New()
	logger.SetFormatter(daemonLogger{
		daemonName: typeName,
		formatter:  logrus.StandardLogger().Formatter,
	})

	defer func() {
		if r := recover(); r != nil {
			logger.WithField("recover", r).Error("Recovered")
		}
	}()

	err := handler.Run(logger)
	if err != nil {
		logger.WithError(err).Error("Failed to run handler")
	}
}

type ScanHandler struct {
	scanner scan.Scanner
}

func (s *ScanHandler) WithScanner(scanner scan.Scanner) *ScanHandler {
	s.scanner = scanner

	return s
}

func (s *ScanHandler) Run(logger *logrus.Logger) error {
	imagePaths, err := s.scanner.Scan()
	if err != nil {
		return err
	}

	if len(imagePaths) == 0 {
		return nil
	}

	logger.WithField("images", len(imagePaths)).Info("Scanned")

	tmpZip, err := makeTmpZipFile(imagePaths)
	if err != nil {
		return err
	}

	queueScans.EnqueueFilePath(tmpZip)
	return nil
}

func makeTmpZipFile(contentFiles []string) (string, error) {
	tmpFile, err := os.CreateTemp("", "scanner-tool-*.zip")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	err = zipFiles(tmpFile, contentFiles)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func zipFiles(zipFile *os.File, contentFiles []string) error {
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range contentFiles {
		fileReader, err := os.Open(file)
		if err != nil {
			return err
		}
		defer fileReader.Close()

		fileContent, err := io.ReadAll(fileReader)
		if err != nil {
			return err
		}

		fileWriter, err := zipWriter.Create(path.Base(file))
		if err != nil {
			return err
		}

		_, err = fileWriter.Write(fileContent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ScanHandler) Close() error {
	return nil
}

type ImageMirrorHandler struct {
}

func (i *ImageMirrorHandler) Run(logger *logrus.Logger) error {
	file, err := queueScans.Dequeue()
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

	err = forAllFilesInZip(file, func(f *zip.File) error {
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
	os.Remove(file.Name())
	return queueRotated.EnqueueFilePath(resultZipFile.Name())
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

type TesseractHandler struct {
}

func (t *TesseractHandler) Run(logger *logrus.Logger) error {
	file, err := queueRotated.Dequeue()
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
	err = forAllFilesInZip(file, func(f *zip.File) error {
		pdfFile, err := zipWriter.Create(pdfFileName(f.Name))
		if err != nil {
			return err
		}

		fileHandle, err := f.Open()
		if err != nil {
			return err
		}
		defer fileHandle.Close()

		err = tesseract.ConvertImageToPdf(fileHandle, pdfFile)
		if err != nil {
			logger.WithError(err).Error("Failed to convert image to pdf")
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	zipWriter.Close()
	os.Remove(file.Name())
	return queueTesseracted.EnqueueFilePath(resultZipFile.Name())
}

func pdfFileName(fileName string) string {
	return strings.TrimSuffix(fileName, ".png") + ".pdf"
}

func (t *TesseractHandler) Close() error {
	return nil
}

type MergeHandler struct {
}

func (m *MergeHandler) Run(logger *logrus.Logger) error {
	file, err := queueTesseracted.Dequeue()
	if err != nil {
		return err
	}
	if file == nil {
		return nil
	}
	defer file.Close()

	var tmpFiles []string
	defer func() {
		for _, tmpFile := range tmpFiles {
			os.Remove(tmpFile)
		}
	}()

	err = forAllFilesInZip(file, func(f *zip.File) error {
		fileHandle, err := f.Open()
		if err != nil {
			return err
		}
		defer fileHandle.Close()

		tmpFile, err := os.CreateTemp("", "scanner-tool-*.pdf")
		if err != nil {
			return err
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, fileHandle)
		if err != nil {
			return err
		}

		tmpFiles = append(tmpFiles, tmpFile.Name())
		return nil
	})
	if err != nil {
		return err
	}

	tmpMergedFile, err := os.CreateTemp("", "scanner-tool-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmpMergedFile.Name())
	defer tmpMergedFile.Close()

	readers := make([]io.ReadSeeker, 0, len(tmpFiles))
	defer func() {
		for _, tmpFile := range readers {
			if r, ok := tmpFile.(io.Closer); ok {
				r.Close()
			}
		}
	}()

	for _, tmpFile := range tmpFiles {
		file, err := os.Open(tmpFile)
		if err != nil {
			return err
		}
		readers = append(readers, file)
	}

	err = api.MergeRaw(readers, tmpMergedFile, false, model.NewDefaultConfiguration())
	if err != nil {
		return err
	}

	os.Remove(file.Name())
	return queueMerged.EnqueueFilePath(tmpMergedFile.Name())
}

func (m *MergeHandler) Close() error {
	return nil
}

type UploadHandler struct {
	cifs *filesystem.Cifs
}

func (u *UploadHandler) WithCifs(cifs *filesystem.Cifs) *UploadHandler {
	u.cifs = cifs

	return u
}

func (u *UploadHandler) Run(logger *logrus.Logger) error {
	file, err := queueMerged.Dequeue()
	if err != nil {
		return err
	}
	if file == nil {
		return nil
	}
	defer file.Close()

	err = u.cifs.UploadReader(path.Base(file.Name())+".pdf", file)
	if err != nil {
		return err
	}

	os.Remove(file.Name())
	return nil
}

func (u *UploadHandler) Close() error {
	return nil
}
