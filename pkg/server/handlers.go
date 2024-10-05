package server

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/sirupsen/logrus"
)

func (s *Server) getIndex(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/html")

	groupedPathsSlice, err := s.listDocuments()
	if err != nil {
		return err
	}

	tpl(w, "index.html", map[string]interface{}{
		"Paths": groupedPathsSlice,
	})

	return nil
}

func (s *Server) getStaticFile(w http.ResponseWriter, r *http.Request) error {
	fileName := r.PathValue("file")

	f, err := publicFs.ReadFile(fmt.Sprintf("public/%s", fileName))
	if err != nil {
		return errNotFound
	}

	mimeType := mime.TypeByExtension(path.Ext(fileName))
	var contentType string
	if mimeType != "" {
		contentType = mimeType
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(f)

	return nil
}

func (s *Server) getFile(w http.ResponseWriter, r *http.Request) error {
	filePath := r.PathValue("file")

	fileContent, err := s.cifs.Download(filePath)
	if err != nil {
		if path.Ext(filePath) == ".png" {
			filePath = "public/preview-image-not-found.svg"
			fileContent, _ = publicFs.ReadFile("public/preview-image-not-found.svg")
		} else {
			return err
		}
	}

	mimeType := mime.TypeByExtension(path.Ext(filePath))
	var contentType string
	if mimeType != "" {
		contentType = mimeType
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(fileContent)

	return nil
}

func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request) error {
	pdfPath := r.PathValue("pdfPath")
	pngPath := strings.TrimSuffix(pdfPath, ".pdf") + ".png"

	_ = s.cifs.Delete(pdfPath, pngPath)

	w.Write(nil)

	return nil
}

func (s *Server) postCombine(w http.ResponseWriter, r *http.Request) error {
	r.ParseForm()
	filename := r.FormValue("filename")

	if filename == "" {
		return fmt.Errorf("filename is required")
	}
	if !strings.HasSuffix(filename, ".pdf") {
		filename += ".pdf"
	}

	filePaths := selectedFiles(r)
	allFiles := []io.ReadSeeker{}
	for _, p := range filePaths {
		if strings.HasSuffix(p, ".pdf") {
			fileContent, err := s.cifs.Download(p)
			if err != nil {
				return err
			}

			allFiles = append(allFiles, newBytesReadSeeker(fileContent))
		}
	}

	output := bytes.NewBuffer(nil)
	err := api.MergeRaw(allFiles, output, false, model.NewDefaultConfiguration())
	if err != nil {
		return err
	}

	err = s.cifs.Upload(filename, output.Bytes())
	if err != nil {
		return err
	}

	// delete all pdfs except the combined one
	for _, p := range filePaths {
		if p != filename {
			s.cifs.Delete(p)
		}
	}

	// delete all pngs
	for _, p := range filePaths {
		s.cifs.Delete(strings.TrimSuffix(p, ".pdf") + ".png")
	}

	paths, err := s.listDocuments()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html")

	tpl(w, "previews.html", map[string]interface{}{
		"Paths": paths,
	})

	return nil
}

func (s *Server) listDocuments() ([]DocumentPath, error) {
	filePaths, err := s.cifs.List()
	if err != nil {
		return nil, err
	}

	// ignore non png and pdf files
	var filteredFilePaths []string
	for _, p := range filePaths {
		if strings.HasSuffix(p, ".pdf") || strings.HasSuffix(p, ".png") {
			filteredFilePaths = append(filteredFilePaths, p)
		}
	}
	filePaths = filteredFilePaths

	groupedPaths := map[string]DocumentPath{}
	for _, p := range filePaths {
		group := strings.TrimSuffix(p, path.Ext(p))
		if _, ok := groupedPaths[group]; !ok {
			groupedPaths[group] = DocumentPath{}
		}

		if strings.HasSuffix(p, ".pdf") {
			groupedPaths[group] = DocumentPath{
				Pdf:   p,
				Image: groupedPaths[group].Image,
			}
		} else if strings.HasSuffix(p, ".png") {
			groupedPaths[group] = DocumentPath{
				Pdf:   groupedPaths[group].Pdf,
				Image: p,
			}
		}
	}

	groupedPathsSlice := DocumentPathList{}
	for _, p := range groupedPaths {
		if p.Image == "" {
			p.Image = strings.TrimSuffix(p.Pdf, ".pdf") + ".png"
		}
		groupedPathsSlice = append(groupedPathsSlice, p)
	}

	// order by pdf
	sort.Sort(groupedPathsSlice)

	return groupedPathsSlice, nil
}

func (s *Server) postReverseOrder(w http.ResponseWriter, r *http.Request) error {
	filePaths := selectedFiles(r)

	for i := 0; i < len(filePaths)/2; i++ {
		pdfA := filePaths[i]
		pdfB := filePaths[len(filePaths)-i-1]
		pngA := strings.TrimSuffix(pdfA, ".pdf") + ".png"
		pngB := strings.TrimSuffix(pdfB, ".pdf") + ".png"

		s.cifs.SwapFileNames(pdfA, pdfB)
		s.cifs.SwapFileNames(pngA, pngB)

		unique := time.Now()
		s.cifs.MakeUnique(pdfA, unique)
		s.cifs.MakeUnique(pdfB, unique)
		s.cifs.MakeUnique(pngA, unique)
		s.cifs.MakeUnique(pngB, unique)
	}

	paths, err := s.listDocuments()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html")

	tpl(w, "previews.html", map[string]interface{}{
		"Paths": paths,
	})

	return nil
}

func selectedFiles(r *http.Request) []string {
	var selectedFiles []string
	r.ParseForm()
	for k := range r.Form {
		if strings.HasPrefix(k, "selected[") {
			if r.FormValue(k) == "on" {
				fileName := strings.TrimSuffix(strings.TrimPrefix(k, "selected["), "]")
				selectedFiles = append(selectedFiles, fileName)
			}
		}
	}

	var selectedDocuments DocumentPathList
	for _, p := range selectedFiles {
		selectedDocuments = append(selectedDocuments, DocumentPath{
			Pdf:   p,
			Image: strings.TrimSuffix(p, ".pdf") + ".png",
		})
	}

	sort.Sort(selectedDocuments)

	selectedFiles = []string{}
	for _, p := range selectedDocuments {
		selectedFiles = append(selectedFiles, p.Pdf)
	}

	return selectedFiles
}

func (s *Server) postSplit(w http.ResponseWriter, r *http.Request) error {
	pdfPath := r.PathValue("pdfPath")

	pdfContent, err := s.cifs.Download(pdfPath)
	if err != nil {
		return err
	}

	pdfContentBuffer := newBytesReadSeeker(pdfContent)
	tmpDir, err := os.MkdirTemp(os.TempDir(), "pdf-split-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	pages, err := api.PageCount(pdfContentBuffer, model.NewDefaultConfiguration())
	if err != nil {
		return err
	}

	var selectedPages []string
	for i := 1; i <= pages; i++ {
		selectedPages = append(selectedPages, fmt.Sprintf("%d", i))
	}

	err = api.ExtractPages(pdfContentBuffer, tmpDir, path.Base(pdfPath), selectedPages, model.NewDefaultConfiguration())
	if err != nil {
		return err
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		return err
	}

	if len(files) != len(selectedPages) {
		return fmt.Errorf("expected %d files, got %d", len(selectedPages), len(files))
	}

	for _, f := range files {
		pdfFile := path.Join(tmpDir, f.Name())
		fileContent, err := os.ReadFile(pdfFile)
		if err != nil {
			return err
		}

		fileId, err := s.cifs.NextFileId()
		if err != nil {
			return err
		}

		err = s.cifs.Upload(fmt.Sprintf("%s%02d.pdf", filePrefix, fileId), fileContent)
		if err != nil {
			return err
		}

		pageImage, err := extractFirstPdfPageAsImage(pdfFile)
		if err != nil {
			return err
		}

		imageBuffer := new(bytes.Buffer)
		err = imageEncoder(imageBuffer, pageImage)
		if err != nil {
			return err
		}

		err = s.cifs.Upload(fmt.Sprintf("%s%02d.png", filePrefix, fileId), imageBuffer.Bytes())
		if err != nil {
			return err
		}
	}

	_ = s.cifs.Delete(pdfPath)
	_ = s.cifs.Delete(strings.TrimSuffix(pdfPath, ".pdf") + ".png")

	paths, err := s.listDocuments()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html")

	tpl(w, "previews.html", map[string]interface{}{
		"Paths": paths,
	})

	return nil
}

func (s *Server) postRefresh(w http.ResponseWriter, r *http.Request) error {
	pdfPath := r.PathValue("pdfPath")
	pngPath := strings.TrimSuffix(pdfPath, ".pdf") + ".png"

	logrus.WithField("pdfPath", pdfPath).Info("Refreshing preview")

	pdfContent, err := s.cifs.Download(pdfPath)
	if err != nil {
		return err
	}

	pageImage, err := extractFirstPdfPageAsImageFromReader(newBytesReadSeeker(pdfContent))
	if err != nil {
		return err
	}

	imageBuffer := new(bytes.Buffer)
	err = imageEncoder(imageBuffer, pageImage)
	if err != nil {
		return err
	}

	err = s.cifs.Upload(pngPath, imageBuffer.Bytes())
	if err != nil {
		return err
	}

	unique := time.Now()
	newPngPath, _ := s.cifs.MakeUnique(pngPath, unique)
	newPdfPath, _ := s.cifs.MakeUnique(pdfPath, unique)

	w.Header().Set("Content-Type", "text/html")

	tpl(w, "previews.html", map[string]interface{}{
		"Paths": []DocumentPath{DocumentPath{
			Pdf:   newPdfPath,
			Image: newPngPath,
		}},
	})

	return nil
}

func extractFirstPdfPageAsImage(pdfLocalPath string) (image.Image, error) {
	f, err := os.Open(pdfLocalPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return extractFirstPdfPageAsImageFromReader(f)
}

func extractFirstPdfPageAsImageFromReader(r io.Reader) (image.Image, error) {
	tmpPngFile := path.Join(os.TempDir(), "tmpPngFile.png")
	defer os.Remove(tmpPngFile)

	command := "pdftoppm"
	args := []string{"-", strings.TrimSuffix(tmpPngFile, ".png"), "-singlefile", "-scale-to", "7026", "-png"}

	logrus.WithField("command", command).WithField("args", args).Info("Executing command")
	cmd := exec.Command(command, args...)
	cmd.Stdin = r
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	imageData, err := os.ReadFile(tmpPngFile)
	if err != nil {
		return nil, err
	}

	out, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, err
	}

	return out, nil
}
