package api

import (
	"bufio"
	"context"
	"encoding/base64"
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
			result.WriteString(fmt.Sprintf("➡ `%s`\n", model.ID))
		}
	}
	return result.String(), nil
}

func HandleUserMessage(bot *tgbotapi.BotAPI, client *http.Client, message *tgbotapi.Message, config *config.Config, u *user.UsageTracker) {
	u.CheckHistory(config.MaxHistorySize-1, config.MaxHistoryTime) // minus one to account for the current message
	u.LastMessageTime = time.Now()

	err := lang.LoadTranslations("./lang/")
	if err != nil {
		log.Fatalf("Error loading translations: %v", err)
	}

	manager, err := configs.NewManager("./config.yaml")
	if err != nil {
		log.Fatalf("Error initializing config manager: %v", err)
	}

	conf := manager.GetConfig()
	errorMessage := lang.Translate("errorText", conf.Lang)
	lastMessageID, stopAnimation := SendLoadingMessage(bot, message.Chat.ID, conf.Lang)
	if lastMessageID == 0 {
		return
	}
	defer func() {
		stopAnimation <- true
	}()

	parts, err := CreateMessageParts(bot, message)
	if err != nil {
		log.Printf("Error creating message parts: %v", err)
		bot.Send(tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, errorMessage))
		return
	}

	u.AddMessage(user.RoleUser, parts...)

	openrouterMessages := buildOpenrouterMessages(u)

	resp, err := CreateChatCompletionStream(context.Background(), client, config, u, openrouterMessages)
	if err != nil {
		fmt.Printf("CreateChatCompletionStream: %v\n", err)
		bot.Send(tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, errorMessage))
		return
	}
	defer resp.Body.Close()

	stopAnimation <- true
	agentMessage, err := ProcessStream(resp, bot, message, lastMessageID)
	if err != nil {
		fmt.Printf("ProcessStream: %v\n", err)
		bot.Send(tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, errorMessage))
		return
	}

	AddAssistantMessageToHistory(u, agentMessage)

	finalText := agentMessage.Text
	if finalText == "" {
		finalText = "Image generated."
	}
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, finalText)
	editMsg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Failed to send final message edit: %v", err)
	}

	u.CurrentStream = nil
}

func CreateMessageParts(bot *tgbotapi.BotAPI, message *tgbotapi.Message) ([]user.MessagePart, error) {
	parts := []user.MessagePart{{Type: user.PartTypeText, Text: message.Text}}

	if message.Photo != nil && len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		fileURL, err := bot.GetFileDirectURL(photo.FileID)
		if err != nil {
			return nil, fmt.Errorf("error getting file URL: %w", err)
		}

		encodedImage, err := downloadAndEncodeImage(fileURL)
		if err != nil {
			return nil, fmt.Errorf("error downloading and encoding image: %w", err)
		}

		parts = append(parts, user.MessagePart{
			Type: user.PartTypeImageURL,
			ImageURL: &user.ImageURL{
				URL: fmt.Sprintf("data:image/jpeg;base64,%s", encodedImage),
			},
		})
	}
	return parts, nil
}

func downloadAndEncodeImage(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(imgBytes), nil
}

func buildOpenrouterMessages(u *user.UsageTracker) []ChatCompletionMessage {
	var openrouterMessages []ChatCompletionMessage
	for _, msg := range u.GetMessages() {
		openrouterMessages = append(openrouterMessages, ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return openrouterMessages
}

func AddAssistantMessageToHistory(u *user.UsageTracker, agentMessage AgentMessage) {
	assistantParts := []user.MessagePart{{Type: user.PartTypeText, Text: agentMessage.Text}}
	for _, img := range agentMessage.Images {
		assistantParts = append(assistantParts, user.MessagePart{
			Type: user.PartTypeImageURL,
			ImageURL: &user.ImageURL{
				URL: fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(img)),
			},
		})
	}
	u.AddMessage(user.RoleAssistant, assistantParts...)
}

type ChatCompletionMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type AgentMessage struct {
	Text   string
	Images [][]byte
}

type StreamResponse struct {
	Choices []struct {
		Delta struct {
			Content interface{} `json:"content"`
			Images  []struct {
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"images"`
		} `json:"delta"`
	} `json:"choices"`
}

func ProcessStream(resp *http.Response, bot *tgbotapi.BotAPI, message *tgbotapi.Message, lastMessageID int) (AgentMessage, error) {
	var agentMessage AgentMessage
	var lastUpdateTime time.Time
	var buffer strings.Builder

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return agentMessage, fmt.Errorf("error reading stream: %w", err)
		}

		buffer.WriteString(line)

		if strings.HasSuffix(buffer.String(), "\n\n") {
			content := buffer.String()
			buffer.Reset()

			lines := strings.Split(strings.TrimSpace(content), "\n")
			for _, l := range lines {
				if strings.HasPrefix(l, "data: ") {
					data := strings.TrimPrefix(l, "data: ")
					if data == "[DONE]" {
						return agentMessage, nil
					}

					var streamResp StreamResponse
					if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
						log.Printf("Error unmarshalling stream response: %v", err)
						continue
					}
					if len(streamResp.Choices) > 0 {
						delta := streamResp.Choices[0].Delta
						if content, ok := delta.Content.(string); ok && content != "" {
							agentMessage.Text += content

							if time.Since(lastUpdateTime) > 500*time.Millisecond && len(agentMessage.Text) > 0 {
								editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, agentMessage.Text)
								editMsg.ParseMode = tgbotapi.ModeMarkdown
								bot.Send(editMsg)
								lastUpdateTime = time.Now()
							}
						}
						if len(delta.Images) > 0 {
							for _, image := range delta.Images {
								if image.ImageURL.URL != "" {
									b64data := image.ImageURL.URL
									if i := strings.Index(b64data, ","); i != -1 {
										b64data = b64data[i+1:]
									}

									imgData, err := base64.StdEncoding.DecodeString(b64data)
									if err != nil {
										log.Printf("Error decoding base64 image: %v", err)
										continue
									}

									agentMessage.Images = append(agentMessage.Images, imgData)

									photoBytes := tgbotapi.FileBytes{
										Name:  "image.png",
										Bytes: imgData,
									}
									photoMsg := tgbotapi.NewPhoto(message.Chat.ID, photoBytes)
									bot.Send(photoMsg)
								}
							}
						}
					}
				}
			}
		}
	}

	return agentMessage, nil
}
