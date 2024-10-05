package server

import (
	"embed"
	"fmt"
	"html/template"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/schidstorm/scanner-tool/pkg/cifs"
	"github.com/schidstorm/scanner-tool/pkg/scan"
)

//go:embed public/*
var publicFs embed.FS

//go:embed templates/*
var templatesFs embed.FS

var errNotFound = fmt.Errorf("not found")

var imageEncoder = png.Encode

type Options struct {
	CifsOptions cifs.Options
	ScanOptions scan.Options
	HttpOptions HttpOptions
}

type HttpOptions struct {
	Addr *string
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
	if d[i].Pdf == d[j].Pdf {
		return d[i].Image < d[j].Image
	}

	return d[i].Pdf < d[j].Pdf
}

func (d DocumentPathList) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

type Server struct {
	scanner scan.Scanner
	cifs    *cifs.Cifs
	daemon  *Daemon
	options Options
}

func NewServer(opts Options) *Server {
	opts = applyDefaults(opts)

	s := &Server{
		options: opts,
	}

	s.scanner = scan.NewScanner(s.options.ScanOptions)
	s.cifs = cifs.NewCifs(s.options.CifsOptions)
	s.daemon = NewDaemon([]DaemonHandler{
		new(ScanHandler).WithScanner(s.scanner),
		new(TesseractHandler).WithCifs(s.cifs),
	})

	return s
}

func applyDefaults(opts Options) Options {
	if opts.HttpOptions.Addr == nil {
		addr := ":8080"
		opts.HttpOptions.Addr = &addr
	}

	return opts
}

func (s *Server) WithScanner(scanner scan.Scanner) *Server {
	s.scanner = scanner
	return s
}

func (s *Server) Start() {
	s.cifs.Start()

	http.HandleFunc("GET /", errHandler(s.getIndex))
	http.HandleFunc("GET /{file}", errHandler(s.getStaticFile))
	http.HandleFunc("GET /file/{file}", errHandler(s.getFile))
	http.HandleFunc("DELETE /delete/{pdfPath}", errHandler(s.deleteFile))
	http.HandleFunc("POST /combine", errHandler(s.postCombine))
	http.HandleFunc("POST /reverseOrder", errHandler(s.postReverseOrder))
	http.HandleFunc("POST /split/{pdfPath}", errHandler(s.postSplit))
	http.HandleFunc("POST /refresh/{pdfPath}", errHandler(s.postRefresh))

	s.daemon.Start()
	go http.ListenAndServe(*s.options.HttpOptions.Addr, nil)
}

func errHandler(handler func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err != nil {
			if r.Header.Get("HX-Request") == "true" {
				htmxError(w, err)
			} else {
				httpError(w, err)
			}
			return
		}
	}
}

func htmxError(w http.ResponseWriter, err error) {
	w.Header().Set("HX-Retarget", "#errors")
	w.Header().Set("HX-Reswap", "afterbegin")

	tpl(w, "error.html", map[string]interface{}{
		"Error": err.Error(),
		"Time":  time.Now(),
	})
}

func httpError(w http.ResponseWriter, err error) {
	if err == errNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) Stop() {
	s.daemon.Stop()
	s.cifs.Stop()
}

func tpl(w io.Writer, filename string, args any) {
	template.Must(template.New(filename).Funcs(template.FuncMap{}).ParseFS(templatesFs, "templates/*.html")).Execute(w, args)
}
