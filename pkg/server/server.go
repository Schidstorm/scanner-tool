package server

import (
	"github.com/schidstorm/scanner-tool/pkg/ai"
	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/filesystem"
	"github.com/schidstorm/scanner-tool/pkg/scan"
)

type Options struct {
	CifsOptions   filesystem.Options `yaml:"cifsoptions"`
	ScanOptions   scan.Options       `yaml:"scanoptions"`
	ChatGptApiKey string             `yaml:"chatgptapikey"`
}

type HttpOptions struct {
	Addr *string
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
	s.daemon = NewDaemon(queueFactory, []DaemonHandler{
		new(ScanHandler).WithScanner(s.scanner),
		new(ImageMirrorHandler),
		new(TesseractHandler),
		new(MergeHandler),
		new(UploadHandler).WithCifs(s.cifs).WithFileNameGuesser(ai.NewChatGPTFileNameGuesser(ai.NewChatGPTClient(s.options.ChatGptApiKey))),
	})

	return s
}

func queueFactory(name string) filequeue.Queue {
	return filequeue.NewFsQueue(name)
}

func (s *Server) Start() error {
	s.cifs.Start()
	s.daemon.Start()
	return nil
}

func (s *Server) Stop() error {
	s.daemon.Stop()
	s.cifs.Close()
	return nil
}
