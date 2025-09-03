package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

func ProcessTextStream(stream *openai.ChatCompletionStream, bot *tgbotapi.BotAPI, message *tgbotapi.Message, lastMessageID int) (string, error) {
	var messageText string
	var responseID string
	var lastSentText string
	var lastSentTime time.Time
	updateInterval := 2 * time.Second

	log.Printf("User: " + message.From.UserName + " Stream response.")

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("\nStream finished, response ID:", responseID)
			return messageText, nil
		}

		if err != nil {
			fmt.Println(err)
			msg := tgbotapi.NewMessage(message.Chat.ID, err.Error())
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)
			return "", err
		}

		log.Printf("%v", response)

		if len(response.Choices) > 0 {
			if response.Choices[0].Delta.Content != "" {
				messageText += response.Choices[0].Delta.Content
				if time.Since(lastSentTime) > updateInterval && messageText != lastSentText {
					editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, lastMessageID, messageText)
					editMsg.ParseMode = tgbotapi.ModeMarkdown
					if _, err := bot.Send(editMsg); err == nil {
						lastSentText = messageText
						lastSentTime = time.Now()
					}
				}
			}
		}
	}
}
