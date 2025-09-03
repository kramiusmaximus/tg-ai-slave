package api

import (
	"fmt"
	"log"
	"openrouter-bot/lang"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendLoadingMessage(bot *tgbotapi.BotAPI, chatID int64, langCode string) (int, chan bool) {
	loadMessage := lang.Translate("loadText", langCode)
	processingMsg := tgbotapi.NewMessage(chatID, loadMessage)
	sentMsg, err := bot.Send(processingMsg)
	if err != nil {
		log.Printf("Failed to send processing message: %v", err)
		return 0, nil
	}
	lastMessageID := sentMsg.MessageID

	stopAnimation := make(chan bool)
	go func() {
		dots := []string{".", "..", "..."}
		i := 0
		for {
			select {
			case <-stopAnimation:
				return
			default:
				time.Sleep(500 * time.Millisecond)
				text := fmt.Sprintf("%s%s", loadMessage, dots[i])
				editMsg := tgbotapi.NewEditMessageText(chatID, lastMessageID, text)
				_, err := bot.Send(editMsg)
				if err != nil {
					log.Printf("Failed to update processing message: %v", err)
				}

				i = (i + 1) % len(dots)
			}
		}
	}()

	return lastMessageID, stopAnimation
}
