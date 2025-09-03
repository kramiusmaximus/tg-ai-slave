package user

import (
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

type UsageTracker struct {
	UserID          string
	UserName        string
	LogsDir         string
	SystemPrompt    string
	LastMessageTime time.Time
	CurrentStream   *openai.ChatCompletionStream
	Usage           *UserUsage
	History         History
	UsageMu         sync.Mutex `json:"-"` // Мьютекс для синхронизации доступа к Usage
	FileMu          sync.Mutex `json:"-"` // Мьютекс для синхронизации доступа к файлу
}

type History struct {
	messages []Message
	mu       sync.Mutex
}

type Message struct {
	Role    string
	Content []MessagePart
}

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"

	PartTypeText     = "text"
	PartTypeImageURL = "image_url"
)

type MessagePart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type UserUsage struct {
	UserName     string    `json:"user_name"`
	UsageHistory UsageHist `json:"usage_history"`
}

type Cost struct {
	Day        float64 `json:"day"`
	Month      float64 `json:"month"`
	AllTime    float64 `json:"all_time"`
	LastUpdate string  `json:"last_update"`
}

type UsageHist struct {
	ChatCost map[string]float64 `json:"chat_cost"`
}

type GenerationResponse struct {
	Data GenerationData `json:"data"`
}

type GenerationData struct {
	ID                     string  `json:"id"`
	Model                  string  `json:"model"`
	Streamed               bool    `json:"streamed"`
	GenerationTime         int     `json:"generation_time"`
	CreatedAt              string  `json:"created_at"`
	TokensPrompt           int     `json:"tokens_prompt"`
	TokensCompletion       int     `json:"tokens_completion"`
	NativeTokensPrompt     int     `json:"native_tokens_prompt"`
	NativeTokensCompletion int     `json:"native_tokens_completion"`
	NumMediaPrompt         int     `json:"num_media_prompt"`
	NumMediaCompletion     int     `json:"num_media_completion"`
	Origin                 string  `json:"origin"`
	TotalCost              float64 `json:"total_cost"`
}
