package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/schidstorm/scanner-tool/pkg/cifs"
	"github.com/schidstorm/scanner-tool/pkg/scan"
	"github.com/schidstorm/scanner-tool/pkg/tesseract"
)

//go:embed public/*
var publicFs embed.FS

//go:embed templates/*
var templatesFs embed.FS

type Options struct {
	CifsOptions cifs.Options
	ScanOptions scan.Options
}

type DocumentPath struct {
	Pdf   string
	Image string
}

type DocumentPathList []DocumentPath

func (d DocumentPathList) Len() int {
	return len(d)
}

func (d DocumentPathList) Less(i, j int) bool {
	return d[i].Pdf < d[j].Pdf
}

func (d DocumentPathList) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

type Server struct {
	scanner scan.Scanner
	cifs    *cifs.Cifs
	options Options
}

func NewServer(opts Options) *Server {
	s := &Server{
		options: opts,
	}

	s.scanner = scan.NewScanner(s.options.ScanOptions)
	s.cifs = cifs.NewCifs(s.options.CifsOptions)

	return s
}

func (s *Server) WithScanner(scanner scan.Scanner) *Server {
	s.scanner = scanner
	return s
}

func (s *Server) Start() {
	s.cifs.Start()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		groupedPathsSlice, err := s.listDocuments()
		if err != nil {
			htmxError(w, err)
		}

		template.Must(template.New("index.html").ParseFS(templatesFs, "templates/*.html")).Execute(w, map[string]interface{}{
			"Paths": groupedPathsSlice,
		})
	})

	http.HandleFunc("/{file}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		fileName := r.PathValue("file")

		contentType := "text/html"
		if strings.HasSuffix(fileName, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(fileName, ".js") {
			contentType = "application/javascript"
		} else if strings.HasSuffix(fileName, ".png") {
			contentType = "image/png"
		}

		w.Header().Set("Content-Type", contentType)

		f, err := publicFs.ReadFile(fmt.Sprintf("public/%s", fileName))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Write(f)
	})

	http.HandleFunc("POST /scan", func(w http.ResponseWriter, r *http.Request) {
		_, err := s.ScanAll()
		if err != nil {
			htmxError(w, err)
		}

		paths, err := s.listDocuments()

		w.Header().Set("Content-Type", "text/html")

		template.Must(template.New("previews.html").ParseFS(templatesFs, "templates/*.html")).Execute(w, map[string]interface{}{
			"Paths": paths,
			"Error": err,
		})
	})

	http.HandleFunc("/file/{file}", func(w http.ResponseWriter, r *http.Request) {
		pdfPath := r.PathValue("file")

		fileContent, err := s.cifs.Download(pdfPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		switch path.Ext(pdfPath) {
		case ".pdf":
			w.Header().Set("Content-Type", "application/pdf")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		w.Write(fileContent)
	})

	http.HandleFunc("DELETE /delete/{pdfPath}", func(w http.ResponseWriter, r *http.Request) {
		pdfPath := r.PathValue("pdfPath")
		pngPath := strings.TrimSuffix(pdfPath, ".pdf") + ".png"

		_ = s.cifs.Delete(pdfPath, pngPath)

		w.Write(nil)
	})

	http.HandleFunc("POST /combine", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		filename := r.FormValue("filename")

		filePaths, err := s.cifs.List()
		if err != nil {
			htmxError(w, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(nil)
			return
		}

		allFiles := []io.ReadSeeker{}
		for _, p := range filePaths {
			if strings.HasSuffix(p, ".pdf") {
				fileContent, err := s.cifs.Download(p)
				if err != nil {
					htmxError(w, err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write(nil)
					return
				}

				allFiles = append(allFiles, newBytesReadSeeker(fileContent))
			}
		}

		output := bytes.NewBuffer(nil)
		err = api.MergeRaw(allFiles, output, false, model.NewDefaultConfiguration())
		if err != nil {
			htmxError(w, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(nil)
			return
		}

		err = s.cifs.Upload(filename, output.Bytes())
		if err != nil {
			htmxError(w, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(nil)
			return
		}

		// delete all pdfs except the combined one
		for _, p := range filePaths {
			if p != filename {
				s.cifs.Delete(p)
			}
		}

		paths, err := s.listDocuments()
		if err != nil {
			htmxError(w, err)
		}

		w.Header().Set("Content-Type", "text/html")

		template.Must(template.New("previews.html").ParseFS(templatesFs, "templates/*.html")).Execute(w, map[string]interface{}{
			"Paths": paths,
		})
	})

	go http.ListenAndServe(":8080", nil)
}

func htmxError(w http.ResponseWriter, err error) {
	type errorHeader struct {
		OnServerError string `json:"onServerError"`
	}
	hxErr := errorHeader{
		OnServerError: err.Error(),
	}
	encoded, _ := json.Marshal(hxErr)
	w.Header().Set("HX-Trigger", string(encoded))
}

func (s *Server) Stop() {
	s.cifs.Stop()
}

func (s *Server) ScanAll() ([]DocumentPath, error) {
	var documentPaths []DocumentPath

	for {
		p, err := s.scan()
		if err != nil {
			if err == scan.ErrNoNewDocuments {
				break
			}

			return nil, err
		}

		documentPaths = append(documentPaths, p)
	}

	return documentPaths, nil
}

func (s *Server) List() ([]string, error) {
	return s.cifs.List()
}

func (s *Server) LoadPdf(pdfPath string) ([]byte, error) {
	return s.cifs.Download(pdfPath)
}

func (s *Server) scan() (DocumentPath, error) {
	imageBuffer := new(bytes.Buffer)

	err := s.scanner.Scan(imageBuffer)
	if err != nil {
		return DocumentPath{}, err
	}

	imageData := imageBuffer.Bytes()
	pdfBuffer := new(bytes.Buffer)
	err = tesseract.ConvertImageToPdf(imageBuffer, pdfBuffer)
	if err != nil {
		return DocumentPath{}, err
	}

	nextId, err := s.nextImageId()
	if err != nil {
		return DocumentPath{}, err
	}

	pdfPath := fmt.Sprintf("scan-%02d.pdf", nextId)
	err = s.cifs.Upload(pdfPath, pdfBuffer.Bytes())
	if err != nil {
		return DocumentPath{}, err
	}

	imagePath := fmt.Sprintf("scan-%02d.png", nextId)
	err = s.cifs.Upload(imagePath, imageData)
	if err != nil {
		return DocumentPath{}, err
	}

	return DocumentPath{
		Pdf:   pdfPath,
		Image: imagePath,
	}, nil
}

func (s *Server) nextImageId() (int, error) {
	files, err := s.cifs.List()
	if err != nil {
		return 0, err
	}

	nextId := 0
	fileNames := map[string]struct{}{}
	for _, f := range files {
		fileNames[f] = struct{}{}
	}

	for i := 0; ; i++ {
		if _, ok := fileNames[fmt.Sprintf("scan-%02d.png", i)]; !ok {
			nextId = i
			break
		}
	}

	return nextId, nil
}

func (s *Server) listDocuments() ([]DocumentPath, error) {
	filePaths, err := s.List()
	if err != nil {
		return nil, err
	}

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
		groupedPathsSlice = append(groupedPathsSlice, p)
	}

	// order by pdf
	sort.Sort(groupedPathsSlice)

	return groupedPathsSlice, nil
}
