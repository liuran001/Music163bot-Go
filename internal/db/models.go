package db

import (
	"time"

	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal"
	"gorm.io/gorm"
)

// SongInfoModel mirrors the existing song_infos schema.
type SongInfoModel struct {
	gorm.Model
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

func (SongInfoModel) TableName() string {
	return "song_infos"
}

func toInternal(model SongInfoModel) *internal.SongInfo {
	return &internal.SongInfo{
		ID:             model.ID,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
		DeletedAt:      deletedAtPtr(model.DeletedAt),
		MusicID:        model.MusicID,
		SongName:       model.SongName,
		SongArtists:    model.SongArtists,
		SongArtistsIDs: model.SongArtistsIDs,
		SongAlbum:      model.SongAlbum,
		AlbumID:        model.AlbumID,
		FileExt:        model.FileExt,
		MusicSize:      model.MusicSize,
		PicSize:        model.PicSize,
		EmbPicSize:     model.EmbPicSize,
		BitRate:        model.BitRate,
		Duration:       model.Duration,
		FileID:         model.FileID,
		ThumbFileID:    model.ThumbFileID,
		FromUserID:     model.FromUserID,
		FromUserName:   model.FromUserName,
		FromChatID:     model.FromChatID,
		FromChatName:   model.FromChatName,
	}
}

func toModel(info *internal.SongInfo) *SongInfoModel {
	if info == nil {
		return &SongInfoModel{}
	}

	model := &SongInfoModel{
		MusicID:        info.MusicID,
		SongName:       info.SongName,
		SongArtists:    info.SongArtists,
		SongArtistsIDs: info.SongArtistsIDs,
		SongAlbum:      info.SongAlbum,
		AlbumID:        info.AlbumID,
		FileExt:        info.FileExt,
		MusicSize:      info.MusicSize,
		PicSize:        info.PicSize,
		EmbPicSize:     info.EmbPicSize,
		BitRate:        info.BitRate,
		Duration:       info.Duration,
		FileID:         info.FileID,
		ThumbFileID:    info.ThumbFileID,
		FromUserID:     info.FromUserID,
		FromUserName:   info.FromUserName,
		FromChatID:     info.FromChatID,
		FromChatName:   info.FromChatName,
	}

	if info.ID != 0 {
		model.ID = info.ID
	}
	if !info.CreatedAt.IsZero() {
		model.CreatedAt = info.CreatedAt
	}
	if !info.UpdatedAt.IsZero() {
		model.UpdatedAt = info.UpdatedAt
	}
	if info.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *info.DeletedAt, Valid: true}
	}

	return model
}

func deletedAtPtr(value gorm.DeletedAt) *time.Time {
	if value.Valid {
		return &value.Time
	}
	return nil
}
