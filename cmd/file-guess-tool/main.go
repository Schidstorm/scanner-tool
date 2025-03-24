package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/schidstorm/scanner-tool/pkg/ai"
	"github.com/schidstorm/scanner-tool/pkg/filesystem"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ChatGptApiKey string             `yaml:"chatgptapikey"`
	CifsOptions   filesystem.Options `yaml:"cifsoptions"`
}

var configFile string

func main() {
	rootCmd := &cobra.Command{
		Use:   "file-guess-tool",
		Short: "A tool to guess file names based on their content",
		Long:  `A tool to guess file names based on their content using AI models.`,
		Run:   cobraRun,
	}

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to the config file")
	rootCmd.MarkPersistentFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		rootCmd.Println(err)
	}
}

func cobraRun(cmd *cobra.Command, args []string) {
	run(args)
}

func run(args []string) {
	cfg, err := readConfigFile(configFile)
	if err != nil {
		logrus.Fatalf("Failed to read config file: %v", err)
	}

	// Initialize the ChatGPT client
	client := ai.NewChatGPTClient(cfg.ChatGptApiKey)
	fileNameGuesser := ai.NewChatGPTFileNameGuesser(client)

	fs, err := initFsystem(cfg.CifsOptions)
	if err != nil {
		panic(err)
	}
	defer fs.Stop()

	files, err := fs.List()
	if err != nil {
		logrus.Fatalf("Failed to list files: %v", err)
	}

	for _, filePath := range files {
		if !strings.HasSuffix(filePath, ".pdf") {
			logrus.Infof("Skipping non-PDF file: %s", filePath)
			continue
		}

		logrus.Info("Processing file: ", filePath)
		pdfData, err := loadPdf(fs, filePath)
		if err != nil {
			logrus.Errorf("Failed to load PDF: %v", err)
			continue
		}
		logrus.Info("Extracting text from PDF")
		text, err := extractTextFromPdf(pdfData)
		if err != nil {
			logrus.Errorf("Failed to extract text from PDF: %v", err)
			continue
		}
		logrus.Info("Guessing file name")
		fileName, err := fileNameGuesser.Guess(text)
		if err != nil {
			logrus.Errorf("Failed to guess file name: %v", err)
			continue
		}
		logrus.Infof("Guessed file name: %s", fileName)

		err = fs.RenameFile(filePath, fileName)
		if err != nil {
			logrus.Errorf("Failed to rename file: %v", err)
			continue
		}
		logrus.Infof("Renamed file %s to %s", filePath, fileName)
	}
}

func initFsystem(cifsOptions filesystem.Options) (*filesystem.Cifs, error) {
	fs := filesystem.NewCifs(cifsOptions)
	go fs.Start()

	return fs, nil
}

func readConfigFile(configFile string) (*Config, error) {
	configFileContent, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(configFileContent, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func loadPdf(fs *filesystem.Cifs, filePath string) ([]byte, error) {
	return fs.Download(filePath)
}

func extractTextFromPdf(pdfData []byte) (string, error) {
	return pdfToText(pdfData)
}

func pdfToText(pdfData []byte) (string, error) {
	cmd := exec.Command("pdftotext", "-", "-")
	inBuffer := bytes.NewBuffer(pdfData)
	outBuffer := &bytes.Buffer{}
	cmd.Stdin = inBuffer
	cmd.Stdout = outBuffer
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outBuffer.String(), nil
}
