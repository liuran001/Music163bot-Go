package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/liuran001/Music163bot-Go/v3/internal"
)

// SearchHandler handles /search and private message search.
type SearchHandler struct {
	Netease internal.NeteaseClient
}

func (h *SearchHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}

	message := update.Message
	threadID := message.MessageThreadID
	replyParams := buildReplyParams(message)
	keyword := commandArguments(message.Text)
	if keyword == "" && message.Chat.Type == "private" {
		keyword = message.Text
	}
	if strings.TrimSpace(keyword) == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          message.Chat.ID,
			Text:            inputKeyword,
			ReplyParameters: &models.ReplyParameters{MessageID: message.ID},
		})
		return
	}

	msgResult, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          message.Chat.ID,
		MessageThreadID: threadID,
		Text:            searching,
		ReplyParameters: replyParams,
	})
	if err != nil {
		return
	}
	if h.Netease == nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msgResult.Chat.ID,
			MessageID: msgResult.ID,
			Text:      noResults,
		})
		return
	}

	searchResult, err := h.Netease.Search(ctx, keyword, 10)
	if err != nil || len(searchResult.Result.Songs) == 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msgResult.Chat.ID,
			MessageID: msgResult.ID,
			Text:      noResults,
		})
		return
	}

	var buttons []models.InlineKeyboardButton
	var textMessage string
	requesterID := int64(0)
	if message.From != nil {
		requesterID = message.From.ID
	}
	for i := 0; i < len(searchResult.Result.Songs) && i < 8; i++ {
		song := searchResult.Result.Songs[i]
		escapedSongName := mdV2Replacer.Replace(song.Name)
		songLink := fmt.Sprintf("[%s](https://music.163.com/song?id=%d)", escapedSongName, song.Id)

		var artistParts []string
		for _, artist := range song.Artists {
			escapedArtist := mdV2Replacer.Replace(artist.Name)
			artistLink := fmt.Sprintf("[%s](https://music.163.com/artist?id=%d)", escapedArtist, artist.Id)
			artistParts = append(artistParts, artistLink)
		}
		songArtists := strings.Join(artistParts, " / ")
		textMessage = fmt.Sprintf("%s%d\\. 「%s」 \\- %s\n", textMessage, i+1, songLink, songArtists)
		buttons = append(buttons, models.InlineKeyboardButton{Text: fmt.Sprintf("%d", i+1), CallbackData: fmt.Sprintf("music %d %d", song.Id, requesterID)})
	}

	keyboard := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{buttons}}
	disablePreview := true
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:             msgResult.Chat.ID,
		MessageID:          msgResult.ID,
		Text:               textMessage,
		ParseMode:          models.ParseModeMarkdown,
		ReplyMarkup:        keyboard,
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: &disablePreview},
	})
}
