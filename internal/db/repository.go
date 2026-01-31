package db

import (
	"context"
	"fmt"
	"time"

	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Repository provides access to the song cache database.
type Repository struct {
	db *gorm.DB
}

// NewSQLiteRepository creates a repository backed by SQLite.
func NewSQLiteRepository(dsn string, gormLogger logger.Interface) (*Repository, error) {
	if dsn == "" {
		return nil, fmt.Errorf("dsn required")
	}

	if gormLogger == nil {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
		Logger:                 gormLogger,
	})
	if err != nil {
		return nil, err
	}

	if err := applySQLitePragmas(db); err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&SongInfoModel{}); err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &Repository{db: db}, nil
}

// FindByMusicID returns a cached song by MusicID.
func (r *Repository) FindByMusicID(ctx context.Context, musicID int) (*internal.SongInfo, error) {
	var model SongInfoModel
	err := r.db.WithContext(ctx).Where("music_id = ?", musicID).First(&model).Error
	if err != nil {
		return nil, err
	}
	return toInternal(model), nil
}

// FindByFileID returns a cached song by FileID.
func (r *Repository) FindByFileID(ctx context.Context, fileID string) (*internal.SongInfo, error) {
	var model SongInfoModel
	err := r.db.WithContext(ctx).Where("file_id = ?", fileID).First(&model).Error
	if err != nil {
		return nil, err
	}
	return toInternal(model), nil
}

// Create inserts a new song record.
func (r *Repository) Create(ctx context.Context, song *internal.SongInfo) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := toModel(song)
		if err := tx.Create(model).Error; err != nil {
			return err
		}
		song.ID = model.ID
		song.CreatedAt = model.CreatedAt
		song.UpdatedAt = model.UpdatedAt
		return nil
	})
}

// Update updates an existing song record.
func (r *Repository) Update(ctx context.Context, song *internal.SongInfo) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := toModel(song)
		return tx.Save(model).Error
	})
}

// Delete removes a song by MusicID.
func (r *Repository) Delete(ctx context.Context, musicID int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Delete(&SongInfoModel{}, "music_id = ?", musicID).Error
	})
}

// Count returns total cached songs.
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SongInfoModel{}).Count(&count).Error
	return count, err
}

// CountByUserID returns cached count by user ID.
func (r *Repository) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SongInfoModel{}).Where("from_user_id = ?", userID).Count(&count).Error
	return count, err
}

// CountByChatID returns cached count by chat ID.
func (r *Repository) CountByChatID(ctx context.Context, chatID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SongInfoModel{}).Where("from_chat_id = ?", chatID).Count(&count).Error
	return count, err
}

// Last returns the last cached record.
func (r *Repository) Last(ctx context.Context) (*internal.SongInfo, error) {
	var model SongInfoModel
	if err := r.db.WithContext(ctx).Last(&model).Error; err != nil {
		return nil, err
	}
	return toInternal(model), nil
}

func applySQLitePragmas(db *gorm.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA cache_size=-64000;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, stmt := range pragmas {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
