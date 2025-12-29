package types

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

type OllamaCloudMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaCloudRequest struct {
	Model    string               `json:"model"`
	Messages []OllamaCloudMessage `json:"messages"`
	Stream   bool                 `json:"stream"`
}

type OllamaCloudResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}
