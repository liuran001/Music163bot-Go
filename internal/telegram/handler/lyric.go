package handler

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/liuran001/Music163bot-Go/v3/internal"
)

// LyricHandler handles /lyric command.
type LyricHandler struct {
	Netease  internal.NeteaseClient
	CacheDir string
}

func (h *LyricHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}
	message := update.Message

	if h.CacheDir == "" {
		h.CacheDir = "./cache"
	}
	ensureDir(h.CacheDir)

	args := commandArguments(message.Text)
	if args == "" && message.ReplyToMessage == nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          message.Chat.ID,
			Text:            inputContent,
			ReplyParameters: &models.ReplyParameters{MessageID: message.ID},
		})
		return
	}

	if args == "" && message.ReplyToMessage != nil {
		args = message.ReplyToMessage.Text
		if args == "" {
			return
		}
	}

	msgResult, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          message.Chat.ID,
		Text:            fetchingLyric,
		ReplyParameters: &models.ReplyParameters{MessageID: message.ID},
	})
	if err != nil {
		return
	}

	musicID := parseMusicID(args)
	if musicID == 0 && h.Netease != nil {
		searchResult, _ := h.Netease.Search(ctx, args, 5)
		if searchResult == nil || len(searchResult.Result.Songs) == 0 {
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID, Text: noResults})
			return
		}
		musicID = searchResult.Result.Songs[0].Id
	}

	if musicID == 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID, Text: noResults})
		return
	}

	if h.Netease == nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID, Text: getLrcFailed})
		return
	}

	lyric, err := h.Netease.GetLyric(ctx, musicID)
	if err != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID, Text: getLrcFailed})
		return
	}

	detail, err := h.Netease.GetSongDetail(ctx, musicID)
	if err != nil || len(detail.Songs) == 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID, Text: getLrcFailed})
		return
	}

	var replacer = strings.NewReplacer("/", " ", "?", " ", "*", " ", ":", " ", "|", " ", "\\", " ", "<", " ", ">", " ", "\"", " ")
	name := fmt.Sprintf("%s - %s.lrc", replacer.Replace(parseArtist(detail.Songs[0])), replacer.Replace(detail.Songs[0].Name))
	lrcPath := fmt.Sprintf("%s/%s", h.CacheDir, name)

	file, err := os.OpenFile(lrcPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID, Text: getLrcFailed})
		return
	}

	write := bufio.NewWriter(file)
	_, _ = write.WriteString(lyric.Lrc.Lyric)
	_ = write.Flush()
	_ = file.Close()
	defer os.Remove(lrcPath)

	fileReader, err := os.Open(lrcPath)
	if err != nil {
		return
	}
	defer fileReader.Close()

	_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID:          message.Chat.ID,
		Document:        &models.InputFileUpload{Filename: filepath.Base(lrcPath), Data: fileReader},
		ReplyParameters: &models.ReplyParameters{MessageID: message.ID},
	})
	if err == nil {
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID})
	}
}
