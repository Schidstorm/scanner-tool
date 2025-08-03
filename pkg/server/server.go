package server

import (
	"github.com/schidstorm/scanner-tool/pkg/ai"
	"github.com/schidstorm/scanner-tool/pkg/filequeue"
	"github.com/schidstorm/scanner-tool/pkg/paperless"
	"github.com/schidstorm/scanner-tool/pkg/scan"
)

type Options struct {
	ScanOptions    scan.Options `yaml:"scanoptions"`
	ChatGptApiKey  string       `yaml:"chatgptapikey"`
	PaperlessToken string       `yaml:"paperlesstoken"`
	PaperlessUrl   string       `yaml:"paperlessurl"`
}

type HttpOptions struct {
	Addr *string
}

type Server struct {
	scanner scan.Scanner
	daemon  *Daemon
	options Options
}

func NewServer(opts Options) *Server {
	s := &Server{
		options: opts,
	}

	s.scanner = scan.NewScanner(s.options.ScanOptions)
	aiInstance := ai.NewChatGPTClient(s.options.ChatGptApiKey)
	s.daemon = NewDaemon(queueFactory, []DaemonHandler{
		new(ScanHandler).WithScanner(s.scanner),
		new(ImageMirrorHandler),
		new(TesseractHandler),
		new(MergeHandler),
		new(AiHandler).WithFileNameGuesser(ai.NewChatGPTFileNameGuesser(aiInstance)).WithFileTagsGuesser(ai.NewChatGPTFileTagsGuesser(aiInstance)),
		new(PaperlessUploadHandler).WithPaperless(paperless.NewPaperless(s.options.PaperlessUrl, s.options.PaperlessToken)),
	})

	return s
}

func queueFactory(name string) filequeue.Queue {
	return filequeue.NewFsQueue(name)
}

func (s *Server) Start() error {
	s.daemon.Start()
	return nil
}

func (s *Server) Stop() error {
	s.daemon.Stop()
	return nil
}
