package handler

import (
	"context"
	"fmt"

	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// RmCacheHandler handles /rmcache command.
type RmCacheHandler struct {
	Repo internal.SongRepository
}

func (h *RmCacheHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil || h.Repo == nil {
		return
	}
	message := update.Message
	args := commandArguments(message.Text)
	if args == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          message.Chat.ID,
			Text:            inputIDorKeyword,
			ReplyParameters: &models.ReplyParameters{MessageID: message.ID},
		})
		return
	}

	musicID := parseMusicID(args)
	if musicID == 0 {
		musicID = parseProgramID(args)
		if musicID != 0 {
			musicID = getProgramRealID(musicID)
		}
	}
	if musicID == 0 {
		return
	}

	songInfo, err := h.Repo.FindByMusicID(ctx, musicID)
	if err == nil && songInfo != nil {
		_ = h.Repo.Delete(ctx, musicID)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          message.Chat.ID,
			Text:            fmt.Sprintf(rmcacheReport, songInfo.SongName),
			ReplyParameters: &models.ReplyParameters{MessageID: message.ID},
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          message.Chat.ID,
		Text:            noCache,
		ReplyParameters: &models.ReplyParameters{MessageID: message.ID},
	})
}
