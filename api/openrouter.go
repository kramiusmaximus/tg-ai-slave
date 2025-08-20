package api

import (
	"context"
	"encoding/json"
	"errors"
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
		// Filter by price
		if model.Pricing.Prompt == "0" {
			// escapedDesc := strings.ReplaceAll(model.Description, "*", "\\*")
			// escapedDesc = strings.ReplaceAll(escapedDesc, "_", "\\_")
			// result.WriteString(fmt.Sprintf("%s - %s\n", model.ID, escapedDesc))
			result.WriteString(fmt.Sprintf("➡ `%s`\n", model.ID))
			// result.WriteString(fmt.Sprintf("➡ `/set_model %s`\n", model.ID))
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

	// Send a loading message with animation points
	loadMessage := lang.Translate("loadText", conf.Lang)
	errorMessage := lang.Translate("errorText", conf.Lang)

	processingMsg := tgbotapi.NewMessage(message.Chat.ID, loadMessage)
	sentMsg, err := bot.Send(processingMsg)
	if err != nil {
		log.Printf("Failed to send processing message: %v", err)
		return ""
	}
	lastMessageID := sentMsg.MessageID

	// Goroutine for animation points
	stopAnimation := make(chan bool)
	go func() {
		dots := []string{"", ".", "..", "...", "..", "."}
		i := 0
		for {
			select {
			case <-stopAnimation:
				return
			default:
				text := fmt.Sprintf("%s%s", loadMessage, dots[i])
				editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, text)
				_, err := bot.Send(editMsg)
				if err != nil {
					log.Printf("Failed to update processing message: %v", err)
				}

				i = (i + 1) % len(dots)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

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
		Stream:           true,
	}

	// Error handling and sending a response message
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Printf("ChatCompletionStream error: %v\n", err)
		stopAnimation <- true
		bot.Send(tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, errorMessage))
		return ""
	}
	defer stream.Close()
	user.CurrentStream = stream

	// Stop the animation when we start receiving a response
	stopAnimation <- true
	var messageText string
	responseID := ""
	log.Printf("User: " + user.UserName + " Stream response. ")

	for {
		response, err := stream.Recv()
		if responseID == "" {
			responseID = response.ID
		}
		if errors.Is(err, io.EOF) {
			fmt.Println("\nStream finished, response ID:", responseID)
			user.AddMessage(openai.ChatMessageRoleUser, message.Text)
			user.AddMessage(openai.ChatMessageRoleAssistant, messageText)
			editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, messageText)
			editMsg.ParseMode = tgbotapi.ModeMarkdown
			_, err := bot.Send(editMsg)
			if err != nil {
				log.Printf("Failed to edit message: %v", err)
			}
			user.CurrentStream = nil
			return responseID
		}

		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			msg := tgbotapi.NewMessage(message.Chat.ID, err.Error())
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)
			user.CurrentStream = nil
			return responseID
		}

		if len(response.Choices) > 0 {
			messageText += response.Choices[0].Delta.Content
			editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, messageText)
			editMsg.ParseMode = tgbotapi.ModeMarkdown
			_, err := bot.Send(editMsg)
			if err != nil {
				continue
			}
		} else {
			log.Printf("Received empty response choices")
			continue
		}
	}
}

func addVisionMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, config *config.Config) openai.ChatCompletionMessage {
	if len(message.Photo) > 0 {
		// Assuming you want the largest photo size
		photoSize := message.Photo[len(message.Photo)-1]
		fileID := photoSize.FileID

		// Download the photo
		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			log.Printf("Error getting file: %v", err)
			return openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: message.Text,
			}
		}

		// Access the file URL
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
