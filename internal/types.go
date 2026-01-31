package internal

import (
	"time"

	"github.com/XiaoMengXinX/Music163Api-Go/types"
)

// SongInfo represents cached song metadata.
// It mirrors the existing schema to keep backward compatibility.
type SongInfo struct {
	ID             uint
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
	MusicID        int
	SongName       string
	SongArtists    string
	SongArtistsIDs string
	SongAlbum      string
	AlbumID        int
	FileExt        string
	MusicSize      int
	PicSize        int
	EmbPicSize     int
	BitRate        int
	Duration       int
	FileID         string
	ThumbFileID    string
	FromUserID     int64
	FromUserName   string
	FromChatID     int64
	FromChatName   string
}

// SongDetail represents NetEase song detail response.
type SongDetail = types.SongsDetailData

// SongURL represents NetEase song URL response.
type SongURL = types.SongsURLData

// SearchResult represents NetEase search response.
type SearchResult = types.SearchSongData

// Lyric represents NetEase lyric response.
type Lyric = types.SongLyricData
