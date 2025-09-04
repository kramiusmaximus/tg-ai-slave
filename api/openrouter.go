package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"openrouter-bot/config"
	configs "openrouter-bot/config"
	"openrouter-bot/lang"
	"openrouter-bot/user"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

type Model struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Pricing     struct {
		Prompt string `json:"prompt"`
	} `json:"pricing"`
}

type APIResponse struct {
	Data []Model `json:"data"`
}

func GetFreeModels() (string, error) {
	manager, err := config.NewManager("./config.yaml")
	if err != nil {
		log.Fatalf("Error initializing config manager: %v", err)
	}
	conf := manager.GetConfig()

	resp, err := http.Get(conf.OpenAIBaseURL + "/models")
	if err != nil {
		return "", fmt.Errorf("error get models: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error read response: %v", err)
	}

	var apiResponse APIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return "", fmt.Errorf("error parse json: %v", err)
	}

	var result strings.Builder
	for _, model := range apiResponse.Data {
		if model.Pricing.Prompt == "0" {
			result.WriteString(fmt.Sprintf("➡ &grave;%s&grave;\n", model.ID))
		}
	}
	return result.String(), nil
}

func HandleChatGPTStreamResponse(bot *tgbotapi.BotAPI, client *openai.Client, message *tgbotapi.Message, config *config.Config, user *user.UsageTracker) string {
	ctx := context.Background()
	user.CheckHistory(config.MaxHistorySize, config.MaxHistoryTime)
	user.LastMessageTime = time.Now()

	err := lang.LoadTranslations("./lang/")
	if err != nil {
		log.Fatalf("Error loading translations: %v", err)
	}

	manager, err := configs.NewManager("./config.yaml")
	if err != nil {
		log.Fatalf("Error initializing config manager: %v", err)
	}

	conf := manager.GetConfig()

	loadMessage := lang.Translate("loadText", conf.Lang)
	errorMessage := lang.Translate("errorText", conf.Lang)

	processingMsg := tgbotapi.NewMessage(message.Chat.ID, loadMessage)
	sentMsg, err := bot.Send(processingMsg)
	if err != nil {
		log.Printf("Failed to send processing message: %v", err)
		return ""
	}
	lastMessageID := sentMsg.MessageID

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: user.SystemPrompt,
		},
	}

	for _, msg := range user.GetMessages() {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if config.Vision == "true" {
		messages = append(messages, addVisionMessage(bot, message, config))
	} else {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: message.Text,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:            config.Model.ModelName,
		FrequencyPenalty: float32(config.Model.FrequencyPenalty),
		PresencePenalty:  float32(config.Model.PresencePenalty),
		Temperature:      float32(config.Model.Temperature),
		TopP:             float32(config.Model.TopP),
		MaxTokens:        config.MaxTokens,
		Messages:         messages,
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		bot.Send(tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, errorMessage))
		return ""
	}

	messageText := resp.Choices[0].Message.Content
	responseID := resp.ID

	user.AddMessage(openai.ChatMessageRoleUser, message.Text)
	user.AddMessage(openai.ChatMessageRoleAssistant, messageText)

	sendChunkedMessage(bot, message.Chat.ID, messageText, lastMessageID)

	return responseID
}

func sendChunkedMessage(bot *tgbotapi.BotAPI, chatID int64, text string, messageID int) {
	if text == "" {
		log.Printf("Warning: sendChunkedMessage called with empty text")
		return
	}
	const chunkSize = 4096
	if len(text) <= chunkSize {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		_, err := bot.Send(editMsg)
		if err != nil {
			log.Printf("Failed to edit message: %v", err)
		}
		return
	}

	runes := []rune(text)
	var from, to int

	// First chunk
	to = chunkSize
	if to > len(runes) {
		to = len(runes)
	}
	lastNewline := strings.LastIndex(string(runes[from:to]), "\n")
	if lastNewline != -1 {
		to = from + lastNewline
	}

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, string(runes[from:to]))
	editMsg.ParseMode = tgbotapi.ModeMarkdown
	_, err := bot.Send(editMsg)
	if err != nil {
		log.Printf("Failed to send initial chunk: %v", err)
		return
	}
	from = to

	// Subsequent chunks
	for from < len(runes) {
		to = from + chunkSize
		if to > len(runes) {
			to = len(runes)
		}
		lastNewline := strings.LastIndex(string(runes[from:to]), "\n")
		if lastNewline != -1 && from+lastNewline < len(runes) {
			to = from + lastNewline
		}

		chunk := string(runes[from:to])
		if chunk == "" {
			continue
		}
		msg := tgbotapi.NewMessage(chatID, chunk)
		msg.ParseMode = tgbotapi.ModeMarkdown
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Failed to send chunk: %v", err)
		}
		from = to
	}
}

func addVisionMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, config *config.Config) openai.ChatCompletionMessage {
	if len(message.Photo) > 0 {
		photoSize := message.Photo[len(message.Photo)-1]
		fileID := photoSize.FileID

		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			log.Printf("Error getting file: %v", err)
			return openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: message.Text,
			}
		}

		fileURL := file.Link(bot.Token)
		fmt.Println("Photo URL:", fileURL)
		if message.Text == "" {
			message.Text = config.VisionPrompt
		}

		return openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleUser,
			MultiContent: []openai.ChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: message.Text,
				},
				{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL:    fileURL,
						Detail: openai.ImageURLDetail(config.VisionDetails),
					},
				},
			},
		}
	} else {
		return openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: message.Text,
		}
	}
}
