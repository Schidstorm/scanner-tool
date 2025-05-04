package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	chatGptUrl             = "https://api.openai.com/v1/responses"
	chatGptModel           = "gpt-4o-mini"
	chatGptMaxOutputTokens = 200
	httpClientTimeout      = 10 * time.Second
)

type ResponsesRequest struct {
	Model           string `json:"model"`
	Input           string `json:"input"`
	Instructions    string `json:"instructions"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	Store           bool   `json:"store"`
}

type ResponsesResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Output []struct {
		Type    string `json:"type"`
		Status  string `json:"status"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

type ChatGPTClient struct {
	apiKey     string
	httpClient *http.Client
}

type httpError struct {
	statusCode int
}

func (e *httpError) Error() string {
	return http.StatusText(e.statusCode)
}

func NewChatGPTClient(apiKey string) *ChatGPTClient {
	httpClient := &http.Client{
		Timeout: httpClientTimeout,
	}

	return &ChatGPTClient{
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

func (c *ChatGPTClient) GenerateResponse(instructions string, prompt string) (string, error) {
	req := ResponsesRequest{
		Model:           chatGptModel,
		Input:           prompt,
		Instructions:    instructions,
		MaxOutputTokens: chatGptMaxOutputTokens,
		Store:           false,
	}

	response, err := c.apiRequest(req)
	if err != nil {
		return "", err
	}
	return response.Output[0].Content[0].Text, nil
}

func (c *ChatGPTClient) apiRequest(req ResponsesRequest) (ResponsesResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return ResponsesResponse{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, chatGptUrl, bytes.NewBuffer(reqBody))
	if err != nil {
		return ResponsesResponse{}, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "scanner-tool")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ResponsesResponse{}, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		readBody, _ := io.ReadAll(resp.Body)
		fmt.Println("Error response body:", string(readBody))
		return ResponsesResponse{}, &httpError{statusCode: resp.StatusCode}
	}

	var response ResponsesResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ResponsesResponse{}, err
	}
	if response.Status != "completed" {
		return ResponsesResponse{}, &httpError{statusCode: http.StatusInternalServerError}
	}

	if len(response.Output) == 0 {
		return ResponsesResponse{}, &httpError{statusCode: http.StatusInternalServerError}
	}
	if len(response.Output[0].Content) == 0 {
		return ResponsesResponse{}, &httpError{statusCode: http.StatusInternalServerError}
	}
	if len(response.Output[0].Content[0].Text) == 0 {
		return ResponsesResponse{}, &httpError{statusCode: http.StatusInternalServerError}
	}
	return response, nil
}
