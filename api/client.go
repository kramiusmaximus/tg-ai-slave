package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"openrouter-bot/config"
	"openrouter-bot/user"
)

type ChatCompletionRequest struct {
	Model            string                  `json:"model"`
	Messages         []ChatCompletionMessage `json:"messages"`
	FrequencyPenalty float32                 `json:"frequency_penalty,omitempty"`
	PresencePenalty  float32                 `json:"presence_penalty,omitempty"`
	Temperature      float32                 `json:"temperature,omitempty"`
	TopP             float32                 `json:"top_p,omitempty"`
	MaxTokens        int                     `json:"max_tokens,omitempty"`
	Stream           bool                    `json:"stream"`
	Modalities       []string                `json:"modalities,omitempty"`
}

func CreateChatCompletionStream(ctx context.Context, client *http.Client, conf *config.Config, user *user.UsageTracker, messages []ChatCompletionMessage) (*http.Response, error) {
	reqBody := ChatCompletionRequest{
		Model:            conf.Model.ModelName,
		Messages:         messages,
		FrequencyPenalty: float32(conf.Model.FrequencyPenalty),
		PresencePenalty:  float32(conf.Model.PresencePenalty),
		Temperature:      float32(conf.Model.Temperature),
		TopP:             float32(conf.Model.TopP),
		MaxTokens:        conf.MaxTokens,
		Stream:           true,
		Modalities:       []string{"image", "text"},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", conf.OpenAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+conf.OpenRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}
