package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/liuran001/Music163bot-Go/v3/internal"
)

// InlineSearchHandler handles inline queries.
type InlineSearchHandler struct {
	Repo    internal.SongRepository
	Netease internal.NeteaseClient
	BotName string
}

func (h *InlineSearchHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.InlineQuery == nil {
		return
	}
	query := update.InlineQuery

	switch {
	case query.Query == "help":
		h.inlineHelp(ctx, b, query)
	case strings.Contains(query.Query, "search"):
		h.inlineSearch(ctx, b, query)
	default:
		musicID := parseMusicID(query.Query)
		if musicID != 0 {
			h.inlineMusic(ctx, b, query, musicID)
		} else {
			h.inlineEmpty(ctx, b, query)
		}
	}
}

func (h *InlineSearchHandler) inlineMusic(ctx context.Context, b *bot.Bot, query *models.InlineQuery, musicID int) {
	if h.Repo == nil {
		h.inlineEmpty(ctx, b, query)
		return
	}
	info, err := h.Repo.FindByMusicID(ctx, musicID)
	if err == nil && info != nil && info.FileID != "" && info.SongName != "" {
		keyboard := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: fmt.Sprintf("%s- %s", info.SongName, info.SongArtists), URL: fmt.Sprintf("https://music.163.com/song?id=%d", info.MusicID)}},
			{{Text: sendMeTo, SwitchInlineQuery: fmt.Sprintf("https://music.163.com/song?id=%d", info.MusicID)}},
		}}

		newAudio := &models.InlineQueryResultCachedDocument{
			ID:             query.ID,
			DocumentFileID: info.FileID,
			Title:          fmt.Sprintf("%s - %s", info.SongArtists, info.SongName),
			Caption:        fmt.Sprintf(musicInfo, info.SongName, info.SongArtists, info.SongAlbum, info.FileExt, float64(info.MusicSize+info.EmbPicSize)/1024/1024, float64(info.BitRate)/1000, h.BotName),
			ReplyMarkup:    keyboard,
			Description:    info.SongAlbum,
		}

		_, _ = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: query.ID,
			Results:       []models.InlineQueryResult{newAudio},
			IsPersonal:    false,
			CacheTime:     3600,
		})
		return
	}

	inlineMsg := &models.InlineQueryResultArticle{
		ID:                  query.ID,
		Title:               noCache,
		Description:         tapToDownload,
		InputMessageContent: &models.InputTextMessageContent{MessageText: query.Query},
	}
	_, _ = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		IsPersonal:    false,
		Results:       []models.InlineQueryResult{inlineMsg},
		CacheTime:     60,
		Button:        &models.InlineQueryResultsButton{Text: tapMeToDown, StartParameter: fmt.Sprintf("%d", musicID)},
	})
}

func (h *InlineSearchHandler) inlineEmpty(ctx context.Context, b *bot.Bot, query *models.InlineQuery) {
	inlineMsg := &models.InlineQueryResultArticle{
		ID:                  query.ID,
		Title:               "输入 help 获取帮助",
		Description:         "Music163bot-Go v2",
		InputMessageContent: &models.InputTextMessageContent{MessageText: "Music163bot-Go v2"},
	}
	_, _ = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		IsPersonal:    false,
		Results:       []models.InlineQueryResult{inlineMsg},
		CacheTime:     3600,
	})
}

func (h *InlineSearchHandler) inlineHelp(ctx context.Context, b *bot.Bot, query *models.InlineQuery) {
	randomID := time.Now().UnixMicro()
	inlineMsg1 := &models.InlineQueryResultArticle{
		ID:                  fmt.Sprintf("%d", randomID),
		Title:               "1.粘贴音乐分享URL或输入MusicID",
		Description:         "Music163bot-Go v2",
		InputMessageContent: &models.InputTextMessageContent{MessageText: "Music163bot-Go v2"},
	}
	inlineMsg2 := &models.InlineQueryResultArticle{
		ID:                  fmt.Sprintf("%d", randomID+1),
		Title:               "2.输入 search+关键词 搜索歌曲",
		Description:         "Music163bot-Go v2",
		InputMessageContent: &models.InputTextMessageContent{MessageText: "Music163bot-Go v2"},
	}
	_, _ = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		IsPersonal:    false,
		Results:       []models.InlineQueryResult{inlineMsg1, inlineMsg2},
		CacheTime:     3600,
	})
}

func (h *InlineSearchHandler) inlineSearch(ctx context.Context, b *bot.Bot, query *models.InlineQuery) {
	keyWord := strings.Replace(query.Query, "search", "", 1)
	keyWord = strings.TrimSpace(keyWord)
	if keyWord == "" {
		inlineMsg := &models.InlineQueryResultArticle{
			ID:                  fmt.Sprintf("%d", time.Now().UnixMicro()),
			Title:               "请输入关键词",
			Description:         "Music163bot-Go v2",
			InputMessageContent: &models.InputTextMessageContent{MessageText: "Music163bot-Go v2"},
		}
		_, _ = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: query.ID,
			IsPersonal:    false,
			Results:       []models.InlineQueryResult{inlineMsg},
			CacheTime:     3600,
		})
		return
	}
	if h.Netease == nil {
		return
	}
	result, err := h.Netease.Search(ctx, keyWord, 10)
	if err != nil || len(result.Result.Songs) == 0 {
		inlineMsg := &models.InlineQueryResultArticle{
			ID:                  fmt.Sprintf("%d", time.Now().UnixMicro()),
			Title:               noResults,
			Description:         noResults,
			InputMessageContent: &models.InputTextMessageContent{MessageText: noResults},
		}
		_, _ = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: query.ID,
			IsPersonal:    false,
			Results:       []models.InlineQueryResult{inlineMsg},
			CacheTime:     3600,
		})
		return
	}
	var inlineMsgs []models.InlineQueryResult
	for i := 0; i < len(result.Result.Songs) && i < 10; i++ {
		var songArtists string
		for idx, artist := range result.Result.Songs[i].Artists {
			if idx == 0 {
				songArtists = artist.Name
			} else {
				songArtists = fmt.Sprintf("%s/%s", songArtists, artist.Name)
			}
		}
		inlineMsg := &models.InlineQueryResultArticle{
			ID:                  fmt.Sprintf("%d", time.Now().UnixMicro()+int64(i)),
			Title:               result.Result.Songs[i].Name,
			Description:         songArtists,
			InputMessageContent: &models.InputTextMessageContent{MessageText: fmt.Sprintf("/netease %d", result.Result.Songs[i].Id)},
		}
		inlineMsgs = append(inlineMsgs, inlineMsg)
	}
	_, _ = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		IsPersonal:    false,
		Results:       inlineMsgs,
		CacheTime:     3600,
	})
}
