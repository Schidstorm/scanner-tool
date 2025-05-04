package ai

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var filetagsGuesserInstructions = "Please guess the letter tags based on the text. Put the tags in a comma-separated list enclosed in parentheses. The tags are used for managing in Paperless. Please provide them in German."
var filetagsGuesserRetries = 3
var filetagsGuesserRetryTimeout = 5 * time.Second

type FileTagsGuesser interface {
	Guess(text string) ([]string, error)
}

type ChatGPTFileTagsGuesser struct {
	client *ChatGPTClient
}

func NewChatGPTFileTagsGuesser(client *ChatGPTClient) *ChatGPTFileTagsGuesser {
	return &ChatGPTFileTagsGuesser{
		client: client,
	}
}

func (g *ChatGPTFileTagsGuesser) Guess(text string) ([]string, error) {
	for i := range filetagsGuesserRetries {
		if i > 0 {
			time.Sleep(filetagsGuesserRetryTimeout)
		}

		fileTags, err := g.guess(text)
		if err != nil {
			logrus.Errorf("Error guessing file tags: %v", err)
			continue
		}

		return fileTags, nil
	}

	return nil, errors.New("failed to guess file tags after retries")
}

func (g *ChatGPTFileTagsGuesser) guess(text string) ([]string, error) {
	resp, err := g.client.GenerateResponse(filetagsGuesserInstructions, text)
	if err != nil {
		return nil, err
	}

	match := regexp.MustCompile(`(\([^\)]+\))`).FindStringSubmatch(resp)
	if len(match) < 2 {
		return nil, errors.New("no taglist found in response")
	}

	tagList := match[1]
	tagList = strings.Trim(tagList, "()")
	tags := strings.Split(tagList, ",")
	var resultTags []string
	for i := range tags {
		tags[i] = strings.Trim(strings.TrimSpace(tags[i]), "'\"")
		if len(tags[i]) > 0 {
			resultTags = append(resultTags, tags[i])
		}
	}

	return resultTags, nil
}
