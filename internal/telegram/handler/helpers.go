package handler

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
	"github.com/XiaoMengXinX/Music163Api-Go/types"
	"github.com/XiaoMengXinX/Music163Api-Go/utils"
	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal"
	"github.com/go-telegram/bot/models"
)

var (
	reg1   = regexp.MustCompile(`(.*)song\?id=`)
	reg2   = regexp.MustCompile("(.*)song/")
	regP1  = regexp.MustCompile(`(.*)program\?id=`)
	regP2  = regexp.MustCompile("(.*)program/")
	regP3  = regexp.MustCompile(`(.*)dj\?id=`)
	regP4  = regexp.MustCompile("(.*)dj/")
	reg5   = regexp.MustCompile("/(.*)")
	reg4   = regexp.MustCompile("&(.*)")
	reg3   = regexp.MustCompile(`\?(.*)`)
	regInt = regexp.MustCompile(`\d+`)
	regUrl = regexp.MustCompile("(http|https)://[\\w\\-_]+(\\.[\\w\\-_]+)+([\\w\\-.,@?^=%&:/~+#]*[\\w\\-@?^=%&/~+#])?")
)

func extractMusicID(message *models.Message) int {
	if message == nil {
		return 0
	}

	text := message.Text
	args := commandArguments(message.Text)
	if args != "" {
		text = args
	}

	text = getRedirectUrl(text)

	if id := parseMusicID(text); id != 0 {
		return id
	}
	if id := parseProgramID(text); id != 0 {
		return getProgramRealID(id)
	}
	return 0
}

func parseArtist(songDetail types.SongDetailData) string {
	var artists string
	for i, ar := range songDetail.Ar {
		if i == 0 {
			artists = ar.Name
		} else {
			artists = fmt.Sprintf("%s/%s", artists, ar.Name)
		}
	}
	return artists
}

func parseMusicID(text string) int {
	var replacer = strings.NewReplacer("\n", "", " ", "")
	messageText := replacer.Replace(getRedirectUrl(text))
	musicUrl := regUrl.FindStringSubmatch(messageText)
	if len(musicUrl) != 0 {
		if strings.Contains(musicUrl[0], "song") {
			ur, _ := url.Parse(musicUrl[0])
			id := ur.Query().Get("id")
			if musicid, _ := strconv.Atoi(id); musicid != 0 {
				return musicid
			}
		}
	}
	musicid, _ := strconv.Atoi(linkTestMusic(messageText))
	return musicid
}

func parseProgramID(text string) int {
	var replacer = strings.NewReplacer("\n", "", " ", "")
	messageText := replacer.Replace(text)
	programid, _ := strconv.Atoi(linkTestProgram(messageText))
	return programid
}

func extractInt(text string) string {
	matchArr := regInt.FindStringSubmatch(text)
	if len(matchArr) == 0 {
		return ""
	}
	return matchArr[0]
}

func linkTestMusic(text string) string {
	return extractInt(reg5.ReplaceAllString(reg4.ReplaceAllString(reg3.ReplaceAllString(reg2.ReplaceAllString(reg1.ReplaceAllString(text, ""), ""), ""), ""), ""))
}

func linkTestProgram(text string) string {
	return extractInt(reg5.ReplaceAllString(reg4.ReplaceAllString(reg3.ReplaceAllString(regP4.ReplaceAllString(regP3.ReplaceAllString(regP2.ReplaceAllString(regP1.ReplaceAllString(text, ""), ""), ""), ""), ""), ""), ""))
}

func getProgramRealID(programID int) int {
	programDetail, err := api.GetProgramDetail(utils.RequestData{}, programID)
	if err != nil {
		return 0
	}
	if programDetail.Program.MainSong.ID != 0 {
		return programDetail.Program.MainSong.ID
	}
	return 0
}

func getRedirectUrl(text string) string {
	var replacer = strings.NewReplacer("\n", "", " ", "")
	messageText := replacer.Replace(text)
	musicUrl := regUrl.FindStringSubmatch(messageText)
	if len(musicUrl) != 0 {
		if strings.Contains(musicUrl[0], "163cn.tv") {
			urlStr := musicUrl[0]
			req, err := http.NewRequest("GET", urlStr, nil)
			if err != nil {
				return text
			}
			client := &http.Client{
				Timeout: 10 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			resp, err := client.Do(req)
			if err != nil {
				return text
			}
			defer resp.Body.Close()
			return resp.Header.Get("location")
		}
	}
	return text
}

func commandArguments(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return ""
	}
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func verifyMD5(filePath string, md5str string) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	md5hash := md5.New()
	if _, err := io.Copy(md5hash, f); err != nil {
		return false, err
	}
	if hex.EncodeToString(md5hash.Sum(nil)) != md5str {
		return false, fmt.Errorf(md5VerFailed)
	}
	return true, nil
}

func ensureDir(path string) {
	_, err := os.Stat(path)
	if err == nil {
		return
	}
	if os.IsNotExist(err) {
		_ = os.MkdirAll(path, os.ModePerm)
	}
}

func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer("/", " ", "?", " ", "*", " ", ":", " ", "|", " ", "\\", " ", "<", " ", ">", " ", "\"", " ")
	return replacer.Replace(name)
}

func cleanupFiles(paths ...string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		_ = os.RemoveAll(p)
	}
}

func buildMusicCaption(songInfo *internal.SongInfo, botName string) string {
	if songInfo == nil {
		return ""
	}
	// Song link
	songNameHTML := fmt.Sprintf("<a href=\"https://music.163.com/song?id=%d\">%s</a>", songInfo.MusicID, songInfo.SongName)

	// Artist links
	artistsHTML := songInfo.SongArtists
	if songInfo.SongArtistsIDs != "" {
		artistIDs := strings.Split(songInfo.SongArtistsIDs, ",")
		artists := strings.Split(songInfo.SongArtists, "/")
		var parts []string
		for i, artist := range artists {
			artist = strings.TrimSpace(artist)
			if i < len(artistIDs) {
				if id, err := strconv.ParseInt(artistIDs[i], 10, 64); err == nil && id > 0 {
					parts = append(parts, fmt.Sprintf("<a href=\"https://music.163.com/artist?id=%d\">%s</a>", id, artist))
					continue
				}
			}
			parts = append(parts, artist)
		}
		artistsHTML = strings.Join(parts, " / ")
	}

	albumHTML := songInfo.SongAlbum
	if songInfo.AlbumID > 0 {
		albumHTML = fmt.Sprintf("<a href=\"https://music.163.com/album?id=%d\">%s</a>", songInfo.AlbumID, songInfo.SongAlbum)
	}

	return fmt.Sprintf("<b>「%s」- %s</b>\n专辑: %s\n<blockquote>%.2fMB %.2fkbps\n#网易云音乐 #%s\n</blockquote>via @%s",
		songNameHTML,
		artistsHTML,
		albumHTML,
		float64(songInfo.MusicSize+songInfo.EmbPicSize)/1024/1024,
		float64(songInfo.BitRate)/1000,
		songInfo.FileExt,
		botName,
	)
}
