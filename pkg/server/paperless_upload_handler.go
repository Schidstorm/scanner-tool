package server

import (
	"bytes"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/schidstorm/scanner-tool/pkg/paperless"
	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/sirupsen/logrus"
)

type PaperlessUploadHandler struct {
	paperless *paperless.Paperless
}

func (u *PaperlessUploadHandler) WithPaperless(paperless *paperless.Paperless) *PaperlessUploadHandler {
	u.paperless = paperless
	return u
}

func (u *PaperlessUploadHandler) Run(logger *logrus.Logger, input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) (resErr error) {
	for f := range input {
		rc, err := f.Open()
		if err != nil {
			logrus.Errorf("Failed to open file %s: %v", f.FileInfo().Name(), err)
			return err
		}
		defer rc.Close()

		pdfData, err := io.ReadAll(rc)
		if err != nil {
			logrus.Errorf("Failed to read file: %v", err)
			return err
		}

		fileReader := bytes.NewReader(pdfData)
		tags := strings.Split(f.Metadata()["tags"], ",")
		tags = mapT(tags, strings.TrimSpace)
		tags = mapT(tags, strings.ToLower)
		sort.Strings(tags)
		tags = slices.Compact(tags)
		tags = filterT(tags, func(tag string) bool {
			return tag != ""
		})

		err = u.paperless.Upload(fileReader, paperless.UploadOptions{
			Title: f.FileInfo().Name(),
			Tags:  tags,
		})

		if err != nil {
			logrus.Errorf("Failed to upload file to Paperless: %v", err)
			return err
		}
	}

	return nil
}

func mapT[T any](input []T, mapper func(T) T) []T {
	result := make([]T, len(input))
	for i, v := range input {
		result[i] = mapper(v)
	}
	return result
}

func filterT[T any](input []T, filter func(T) bool) []T {
	result := make([]T, 0, len(input))
	for _, v := range input {
		if filter(v) {
			result = append(result, v)
		}
	}
	return result
}

func (u *PaperlessUploadHandler) Close() error {
	return nil
}
