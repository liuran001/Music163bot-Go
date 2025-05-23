package bot

import (
	"fmt"
	"strings"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func processSearch(message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	var msgResult tgbotapi.Message
	var keyword string
	if message.Chat.IsPrivate() && !message.IsCommand() {
		keyword = message.Text
	} else {
		keyword = message.CommandArguments()
	}

	if keyword == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, inputKeyword)
		msg.ReplyToMessageID = message.MessageID
		msgResult, err = bot.Send(msg)
		return err
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, searching)
	msg.ReplyToMessageID = message.MessageID
	msgResult, err = bot.Send(msg)
	if err != nil {
		return err
	}
	searchResult, _ := api.SearchSong(data, api.SearchSongConfig{
		Keyword: keyword,
		Limit:   10,
	})
	if len(searchResult.Result.Songs) == 0 {
		newEditMsg := tgbotapi.NewEditMessageText(message.Chat.ID, msgResult.MessageID, noResults)
		msgResult, err = bot.Send(newEditMsg)
		return err
	}
	var inlineButton []tgbotapi.InlineKeyboardButton
	var textMessage string
	for i := 0; i < len(searchResult.Result.Songs) && i < 8; i++ {
		song := searchResult.Result.Songs[i]
		escapedSongName := mdV2Replacer.Replace(song.Name)
		songLink := fmt.Sprintf("[%s](https://music.163.com/song?id=%d)", escapedSongName, song.Id)

		var songArtistsParts []string
		for _, artist := range song.Artists {
			escapedArtistName := mdV2Replacer.Replace(artist.Name)
			artistLink := fmt.Sprintf("[%s](https://music.163.com/artist?id=%d)", escapedArtistName, artist.Id)
			songArtistsParts = append(songArtistsParts, artistLink)
		}
		songArtists := strings.Join(songArtistsParts, " / ")

		textMessage = fmt.Sprintf("%s%d\\. 「%s」 \\- %s\n", textMessage, i+1, songLink, songArtists)
		inlineButton = append(inlineButton, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d", i+1), fmt.Sprintf("music %d", song.Id)))
	}
	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(inlineButton)
	newEditMsg := tgbotapi.NewEditMessageText(message.Chat.ID, msgResult.MessageID, textMessage)
	newEditMsg.ReplyMarkup = &numericKeyboard
	newEditMsg.ParseMode = tgbotapi.ModeMarkdownV2
	newEditMsg.DisableWebPagePreview = true
	message, err = bot.Send(newEditMsg)
	if err != nil {
		return err
	}
	return err
}
