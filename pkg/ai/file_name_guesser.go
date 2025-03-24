package ai

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var instructions = "Please guess the file name based on the text. Put the filename in single quotes. The filename should be good for sorting and beginn with the date. Please do it in german."
var retries = 3
var retryTimeout = 5 * time.Second

type FileNameGuesser interface {
	Guess(text string) (string, error)
}

type ChatGPTFileNameGuesser struct {
	client *ChatGPTClient
}

func NewChatGPTFileNameGuesser(client *ChatGPTClient) *ChatGPTFileNameGuesser {
	return &ChatGPTFileNameGuesser{
		client: client,
	}
}

func (g *ChatGPTFileNameGuesser) Guess(text string) (string, error) {
	for i := range retries {
		if i > 0 {
			time.Sleep(retryTimeout)
		}

		fileName, err := g.guess(text)
		if err != nil {
			logrus.Errorf("Error guessing file name: %v", err)
			continue
		}

		if strings.Contains(fileName, " ") {
			logrus.Warnf("File name contains spaces: %s", fileName)
			continue
		}

		return fileName, nil
	}

	return "", errors.New("failed to guess file name after retries")
}

func (g *ChatGPTFileNameGuesser) guess(text string) (string, error) {
	resp, err := g.client.GenerateResponse(instructions, text)
	if err != nil {
		return "", err
	}

	match := regexp.MustCompile(`('|")([^'"]+)('|")`).FindStringSubmatch(resp)
	if len(match) < 2 {
		return "", errors.New("no filename found in response")
	}

	fileName := match[2]
	if len(fileName) > 255 {
		return "", errors.New("filename is too long")
	}
	if len(fileName) == 0 {
		return "", errors.New("filename is empty")
	}

	fileName = strings.TrimSuffix(fileName, ".txt")
	fileName = strings.TrimSuffix(fileName, ".csv")

	if !strings.HasSuffix(fileName, ".pdf") {
		fileName += ".pdf"
	}

	return fileName, nil
}
