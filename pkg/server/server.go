package server

import (
	"github.com/schidstorm/scanner-tool/pkg/filesystem"
	"github.com/schidstorm/scanner-tool/pkg/scan"
)

type Options struct {
	CifsOptions filesystem.Options
	ScanOptions scan.Options
}

type HttpOptions struct {
	Addr *string
}

type DocumentGroup struct {
	Name  string
	Paths DocumentPathList
}

type DocumentGroupList []DocumentGroup

func (d DocumentGroupList) Len() int {
	return len(d)
}

func (d DocumentGroupList) Less(i, j int) bool {
	return d[i].Name < d[j].Name
}

func (d DocumentGroupList) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
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
	cifs    *filesystem.Cifs
	daemon  *Daemon
	options Options
}

func NewServer(opts Options) *Server {
	s := &Server{
		options: opts,
	}

	s.scanner = scan.NewScanner(s.options.ScanOptions)
	s.cifs = filesystem.NewCifs(s.options.CifsOptions)
	s.daemon = NewDaemon([]DaemonHandler{
		new(ScanHandler).WithScanner(scan.NewScanner(s.options.ScanOptions)),
		new(ImageMirrorHandler),
		new(TesseractHandler),
		new(MergeHandler),
		new(UploadHandler).WithCifs(filesystem.NewCifs(s.options.CifsOptions)),
	})

	return s
}

func (s *Server) Start() error {
	s.daemon.Start()
	return nil
}

func (s *Server) Stop() error {
	s.daemon.Stop()
	s.cifs.Close()
	return nil
}
