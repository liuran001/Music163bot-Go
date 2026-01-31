package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	marker "github.com/XiaoMengXinX/163KeyMarker"
	"github.com/XiaoMengXinX/Music163Api-Go/types"
	downloader "github.com/XiaoMengXinX/SimpleDownloader"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/liuran001/Music163bot-Go/v3/internal"
)

// MusicHandler handles /music and related commands.
type MusicHandler struct {
	Repo            internal.SongRepository
	Netease         internal.NeteaseClient
	Pool            internal.WorkerPool
	Logger          internal.Logger
	CacheDir        string
	BotName         string
	CheckMD5        bool
	DownloadTimeout time.Duration
	ReverseProxy    string
	Limiter         chan struct{}
}

// Handle processes music download and send flow.
func (h *MusicHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}
	message := update.Message

	musicID := extractMusicID(message)
	if musicID == 0 {
		return
	}

	_ = h.processMusic(ctx, b, message, musicID)
}

func (h *MusicHandler) processMusic(ctx context.Context, b *bot.Bot, message *models.Message, musicID int) error {
	if h.CacheDir == "" {
		h.CacheDir = "./cache"
	}
	ensureDir(h.CacheDir)
	threadID := 0
	if message != nil {
		threadID = message.MessageThreadID
	}
	replyParams := buildReplyParams(message)

	if h.Limiter == nil {
		h.Limiter = make(chan struct{}, 4)
	}

	var songInfo internal.SongInfo
	var msgResult *models.Message

	sendFailed := func(err error) {
		var errText string
		if strings.Contains(fmt.Sprintf("%v", err), md5VerFailed) || strings.Contains(fmt.Sprintf("%v", err), downloadTimeout) {
			errText = "%v"
		} else {
			errText = uploadFailed
		}
		text := fmt.Sprintf(musicInfoMsg+errText, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024, strings.ReplaceAll(err.Error(), "BOT_TOKEN", "BOT_TOKEN"))
		if msgResult != nil {
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    msgResult.Chat.ID,
				MessageID: msgResult.ID,
				Text:      text,
			})
		}
	}

	// Try cache
	if h.Repo != nil {
		cached, err := h.Repo.FindByMusicID(ctx, musicID)
		if err == nil && cached != nil {
			songInfo = *cached
			if songInfo.SongArtistsIDs == "" || songInfo.AlbumID == 0 {
				if h.Netease != nil {
					if detail, err := h.Netease.GetSongDetail(ctx, musicID); err == nil && len(detail.Songs) > 0 {
						var artistIDs []string
						for _, ar := range detail.Songs[0].Ar {
							artistIDs = append(artistIDs, fmt.Sprintf("%d", ar.Id))
						}
						songInfo.SongArtistsIDs = strings.Join(artistIDs, ",")
						songInfo.AlbumID = detail.Songs[0].Al.Id
						_ = h.Repo.Update(ctx, &songInfo)
					}
				}
			}

			msgResult, _ = sendStatusMessage(ctx, b, message.Chat.ID, threadID, replyParams, fmt.Sprintf(musicInfoMsg+hitCache, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024))

			if _, err = h.sendMusic(ctx, b, message, &songInfo, "", ""); err != nil {
				sendFailed(err)
				return err
			}
			if msgResult != nil {
				_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID})
			}
			return nil
		}
	}

	msgResult, _ = sendStatusMessage(ctx, b, message.Chat.ID, threadID, replyParams, waitForDown)

	// Limit concurrent downloads
	h.Limiter <- struct{}{}
	defer func() { <-h.Limiter }()

	if h.Repo != nil {
		cached, err := h.Repo.FindByMusicID(ctx, musicID)
		if err == nil && cached != nil {
			songInfo = *cached
			if msgResult != nil {
				_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
					ChatID:    msgResult.Chat.ID,
					MessageID: msgResult.ID,
					Text:      fmt.Sprintf(musicInfoMsg+hitCache, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024),
				})
			}
			if _, err = h.sendMusic(ctx, b, message, &songInfo, "", ""); err != nil {
				sendFailed(err)
				return err
			}
			if msgResult != nil {
				_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID})
			}
			return nil
		}
	}

	if msgResult != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msgResult.Chat.ID,
			MessageID: msgResult.ID,
			Text:      fetchInfo,
		})
	}

	if h.Netease == nil {
		return errors.New("netease client not configured")
	}

	songDetail, err := h.Netease.GetSongDetail(ctx, musicID)
	if err != nil || len(songDetail.Songs) == 0 {
		if msgResult != nil {
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    msgResult.Chat.ID,
				MessageID: msgResult.ID,
				Text:      fetchInfoFailed,
			})
		}
		return err
	}

	songURL, err := h.Netease.GetSongURL(ctx, musicID, "")
	if err != nil || len(songURL.Data) == 0 || songURL.Data[0].Url == "" {
		if msgResult != nil {
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    msgResult.Chat.ID,
				MessageID: msgResult.ID,
				Text:      getUrlFailed,
			})
		}
		return err
	}

	fillSongInfo(&songInfo, songDetail, songURL, message)

	if msgResult != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msgResult.Chat.ID,
			MessageID: msgResult.ID,
			Text:      fmt.Sprintf(musicInfoMsg+downloading, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024),
		})
	}

	musicPath, picPath, cleanupList, err := h.downloadAndPrepare(ctx, songDetail.Songs[0], songURL.Data[0], msgResult, b, message, &songInfo)
	if err != nil {
		cleanupFiles(append(cleanupList, musicPath, picPath)...)
		sendFailed(err)
		return err
	}
	defer cleanupFiles(append(cleanupList, musicPath, picPath)...)

	if msgResult != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msgResult.Chat.ID,
			MessageID: msgResult.ID,
			Text:      fmt.Sprintf(musicInfoMsg+uploading, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024),
		})
	}

	audio, err := h.sendMusic(ctx, b, message, &songInfo, musicPath, picPath)
	if err != nil {
		sendFailed(err)
		return err
	}

	if audio.Audio != nil {
		songInfo.FileID = audio.Audio.FileID
		if audio.Audio.Thumbnail != nil {
			songInfo.ThumbFileID = audio.Audio.Thumbnail.FileID
		}
	}

	if h.Repo != nil {
		_ = h.Repo.Create(ctx, &songInfo)
	}

	if msgResult != nil {
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msgResult.Chat.ID, MessageID: msgResult.ID})
	}

	return nil
}

func (h *MusicHandler) downloadAndPrepare(ctx context.Context, detail types.SongDetailData, urlData types.SongURLData, msg *models.Message, b *bot.Bot, message *models.Message, songInfo *internal.SongInfo) (string, string, []string, error) {
	cleanupList := make([]string, 0, 4)
	url := urlData.Url
	baseURL := url
	if idx := strings.Index(url, "?"); idx != -1 {
		baseURL = url[:idx]
	}
	switch path.Ext(path.Base(baseURL)) {
	case ".mp3":
		songInfo.FileExt = "mp3"
	case ".flac":
		songInfo.FileExt = "flac"
	default:
		songInfo.FileExt = "mp3"
	}

	if h.DownloadTimeout > 0 {
		// SimpleDownloader uses seconds
	}

	client := downloader.NewDownloader().SetSavePath(h.CacheDir).SetBreakPoint(true)
	if h.DownloadTimeout > 0 {
		client.SetTimeOut(h.DownloadTimeout)
	} else {
		client.SetTimeOut(60 * time.Second)
	}

	if detail.Al.PicUrl != "" {
		if picRes, err := http.Head(detail.Al.PicUrl); err == nil {
			songInfo.PicSize = int(picRes.ContentLength)
		}
	}

	stamp := time.Now().UnixMicro()
	musicFileName := fmt.Sprintf("%d-%s", stamp, path.Base(url))

	task, _ := client.NewDownloadTask(url)
	hostReplacer := strings.NewReplacer("m8.", "m7.", "m801.", "m701.", "m804.", "m701.", "m704.", "m701.")
	host := task.GetHostName()
	task.ReplaceHostName(hostReplacer.Replace(host)).ForceHttps().ForceMultiThread()
	errCh := task.SetFileName(musicFileName).DownloadWithChannel()

	updateStatus := func(task *downloader.DownloadTask, ch chan error, statusText string) error {
		var lastUpdateTime int64
		for {
			select {
			case err := <-ch:
				return err
			default:
				writtenBytes := task.GetWrittenBytes()
				if task.GetFileSize() == 0 || writtenBytes == 0 || time.Now().Unix()-lastUpdateTime < 5 {
					continue
				}
				if msg != nil {
					_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
						ChatID:    msg.Chat.ID,
						MessageID: msg.ID,
						Text:      fmt.Sprintf(musicInfoMsg+statusText+downloadStatus, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024, task.CalculateSpeed(time.Millisecond*500), float64(writtenBytes)/1024/1024, float64(task.GetFileSize())/1024/1024, (writtenBytes*100)/task.GetFileSize()),
					})
				}
				lastUpdateTime = time.Now().Unix()
			}
		}
	}

	if err := updateStatus(task, errCh, downloading); err != nil {
		if h.ReverseProxy != "" {
			retryCh := task.WithResolvedIpOnHost(h.ReverseProxy).DownloadWithChannel()
			if err := updateStatus(task, retryCh, redownloading); err != nil {
				task.CleanTempFiles()
				return "", "", cleanupList, err
			}
		} else {
			task.CleanTempFiles()
			return "", "", cleanupList, err
		}
	}

	filePath := filepath.Join(h.CacheDir, musicFileName)
	if h.CheckMD5 && urlData.Md5 != "" {
		if ok, _ := verifyMD5(filePath, urlData.Md5); !ok {
			_ = os.Remove(filePath)
			return "", "", cleanupList, fmt.Errorf("%s\n%s", md5VerFailed, retryLater)
		}
	}

	picPath, resizePicPath := "", ""
	if detail.Al.PicUrl != "" {
		p, _ := client.NewDownloadTask(detail.Al.PicUrl)
		_ = p.SetFileName(fmt.Sprintf("%d-%s", stamp, path.Base(detail.Al.PicUrl))).Download()
		picPath = filepath.Join(h.CacheDir, fmt.Sprintf("%d-%s", stamp, path.Base(detail.Al.PicUrl)))
		cleanupList = append(cleanupList, picPath)
		if resized, err := resizeImg(picPath); err == nil {
			resizePicPath = resized
			cleanupList = append(cleanupList, resizePicPath)
		}
	}

	if picPath != "" {
		if stat, err := os.Stat(picPath); err == nil {
			if stat.Size() > 2*1024*1024 {
				picPath = resizePicPath
				if embStat, err := os.Stat(resizePicPath); err == nil {
					songInfo.EmbPicSize = int(embStat.Size())
				}
			} else {
				songInfo.EmbPicSize = songInfo.PicSize
			}
		}
	}

	// Move file to final path
	finalDir := filepath.Join(h.CacheDir, fmt.Sprintf("%d", stamp))
	_ = os.Mkdir(finalDir, os.ModePerm)
	fileName := sanitizeFileName(fmt.Sprintf("%v - %v.%v", strings.ReplaceAll(songInfo.SongArtists, "/", ","), songInfo.SongName, songInfo.FileExt))
	finalPath := filepath.Join(finalDir, fileName)
	if err := os.Rename(filePath, finalPath); err == nil {
		filePath = finalPath
	}
	cleanupList = append(cleanupList, filePath, finalDir)

	// Add ID3
	mark := marker.CreateMarker(detail, urlData)
	file, _ := os.Open(filePath)
	defer file.Close()

	var pic *os.File
	if picPath != "" {
		pic, _ = os.Open(picPath)
		defer pic.Close()
	}

	if err := marker.AddMusicID3V2(file, pic, mark); err != nil {
		file, _ = os.Open(filePath)
		defer file.Close()
		if err := marker.AddMusicID3V2(file, nil, mark); err != nil {
			return "", "", cleanupList, err
		}
	}

	return filePath, picPath, cleanupList, nil
}

func (h *MusicHandler) sendMusic(ctx context.Context, b *bot.Bot, message *models.Message, songInfo *internal.SongInfo, musicPath, picPath string) (*models.Message, error) {
	if songInfo == nil {
		return nil, errors.New("song info required")
	}
	threadID := 0
	if message != nil {
		threadID = message.MessageThreadID
	}

	var audioFile models.InputFile
	if songInfo.FileID != "" {
		audioFile = &models.InputFileString{Data: songInfo.FileID}
	} else {
		file, err := os.Open(musicPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		audioFile = &models.InputFileUpload{Filename: filepath.Base(musicPath), Data: file}
		_, _ = b.SendChatAction(ctx, &bot.SendChatActionParams{ChatID: message.Chat.ID, MessageThreadID: threadID, Action: models.ChatActionUploadDocument})
	}

	caption := buildMusicCaption(songInfo, h.BotName)
	params := &bot.SendAudioParams{
		ChatID:          message.Chat.ID,
		MessageThreadID: threadID,
		Audio:           audioFile,
		Caption:         caption,
		ParseMode:       models.ParseModeHTML,
		Title:           songInfo.SongName,
		Performer:       songInfo.SongArtists,
		Duration:        songInfo.Duration,
		ReplyParameters: buildReplyParams(message),
	}

	if songInfo.ThumbFileID != "" {
		params.Thumbnail = &models.InputFileString{Data: songInfo.ThumbFileID}
	}
	if picPath != "" {
		pic, err := os.Open(picPath)
		if err == nil {
			defer pic.Close()
			params.Thumbnail = &models.InputFileUpload{Filename: filepath.Base(picPath), Data: pic}
		}
	}

	audio, err := b.SendAudio(ctx, params)
	if err != nil && strings.Contains(fmt.Sprintf("%v", err), "replied message not found") {
		params.ReplyParameters = nil
		audio, err = b.SendAudio(ctx, params)
	}
	return audio, err
}

func buildReplyParams(message *models.Message) *models.ReplyParameters {
	if message == nil {
		return nil
	}
	if message.Chat.Type != "private" {
		return nil
	}
	return &models.ReplyParameters{MessageID: message.ID}
}

func sendStatusMessage(ctx context.Context, b *bot.Bot, chatID int64, threadID int, replyParams *models.ReplyParameters, text string) (*models.Message, error) {
	params := &bot.SendMessageParams{
		ChatID:          chatID,
		MessageThreadID: threadID,
		Text:            text,
		ReplyParameters: replyParams,
	}
	msg, err := b.SendMessage(ctx, params)
	if err != nil && replyParams != nil && strings.Contains(fmt.Sprintf("%v", err), "replied message not found") {
		params.ReplyParameters = nil
		msg, err = b.SendMessage(ctx, params)
	}
	return msg, err
}

func fillSongInfo(songInfo *internal.SongInfo, detail *internal.SongDetail, url *internal.SongURL, message *models.Message) {
	songInfo.MusicID = detail.Songs[0].Id
	songInfo.Duration = detail.Songs[0].Dt / 1000
	songInfo.SongName = detail.Songs[0].Name
	songInfo.SongArtists = parseArtist(detail.Songs[0])

	var artistIDs []string
	for _, ar := range detail.Songs[0].Ar {
		artistIDs = append(artistIDs, fmt.Sprintf("%d", ar.Id))
	}
	songInfo.SongArtistsIDs = strings.Join(artistIDs, ",")

	songInfo.SongAlbum = detail.Songs[0].Al.Name
	songInfo.AlbumID = detail.Songs[0].Al.Id
	songInfo.MusicSize = url.Data[0].Size
	songInfo.BitRate = 8 * url.Data[0].Size / (detail.Songs[0].Dt / 1000)

	if message != nil {
		songInfo.FromChatID = message.Chat.ID
		if message.Chat.Type == "private" {
			songInfo.FromChatName = message.Chat.Username
		} else {
			songInfo.FromChatName = message.Chat.Title
		}
		if message.From != nil {
			songInfo.FromUserID = message.From.ID
			songInfo.FromUserName = message.From.Username
		}
	}
}
