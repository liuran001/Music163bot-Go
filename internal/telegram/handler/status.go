package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/liuran001/Music163bot-Go/v3/internal"
)

// StatusHandler handles /status command.
type StatusHandler struct {
	Repo internal.SongRepository
}

var statLimiter = make(chan struct{}, 1)

func (h *StatusHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil || h.Repo == nil {
		return
	}
	message := update.Message

	statLimiter <- struct{}{}
	defer func() {
		time.Sleep(500 * time.Millisecond)
		<-statLimiter
	}()

	fromCount, _ := h.Repo.Count(ctx)
	chatCount, _ := h.Repo.CountByChatID(ctx, message.Chat.ID)
	chatInfo := message.Chat.Title
	if message.Chat.Username != "" && message.Chat.Title == "" {
		chatInfo = fmt.Sprintf("[%s](tg://user?id=%d)", mdV2Replacer.Replace(message.Chat.Username), message.Chat.ID)
	} else if message.Chat.Username != "" {
		chatInfo = fmt.Sprintf("[%s](https://t.me/%s)", mdV2Replacer.Replace(message.Chat.Title), message.Chat.Username)
	} else {
		chatInfo = fmt.Sprintf("%s", mdV2Replacer.Replace(message.Chat.Title))
	}

	userID := int64(0)
	userCount := int64(0)
	if message.From != nil {
		userID = message.From.ID
		userCount, _ = h.Repo.CountByUserID(ctx, userID)
	}

	msgText := fmt.Sprintf(statusInfo, fromCount, chatInfo, chatCount, userID, userID, userCount)
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          message.Chat.ID,
		Text:            msgText,
		ParseMode:       models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{MessageID: message.ID},
	})
}
