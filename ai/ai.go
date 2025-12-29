package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	config "github.com/ashr-tech/csv-migration-tools/config"
	types "github.com/ashr-tech/csv-migration-tools/types"
)

func CallAI(prompt string, mode *string) (string, error) {
	switch *mode {
	case "local":
		return callLocalOllama(prompt)
	default:
		return callCloudOllama(prompt)
	}
}

func callLocalOllama(prompt string) (string, error) {
	reqBody := types.OllamaRequest{
		Model:  config.LOCAL_AI_MODEL,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(config.LOCAL_AI_ENDPOINT, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp types.OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}

func callCloudOllama(prompt string) (string, error) {
	apiKey := os.Getenv("OLLAMA_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OLLAMA_API_KEY is not set")
	}

	reqBody := types.OllamaCloudRequest{
		Model: config.CLOUD_AI_MODEL,
		Messages: []types.OllamaCloudMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		"POST",
		config.CLOUD_AI_ENDPOINT,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf(
			"ollama cloud http %d:\n%s",
			resp.StatusCode,
			string(body),
		)
	}

	var ollamaResp types.OllamaCloudResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf(
			"json parse error: %v\nraw body:\n%s",
			err,
			string(body),
		)
	}

	return ollamaResp.Message.Content, nil
}
