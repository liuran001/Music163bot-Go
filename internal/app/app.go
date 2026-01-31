package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal/config"
	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal/db"
	logpkg "github.com/XiaoMengXinX/Music163bot-Go/v2/internal/logger"
	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal/netease"
	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal/telegram"
	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal/telegram/handler"
	"github.com/XiaoMengXinX/Music163bot-Go/v2/internal/worker"
	gormlogger "gorm.io/gorm/logger"
)

// App wires all application dependencies.
type App struct {
	Config   *config.Config
	Logger   *logpkg.Logger
	DB       *db.Repository
	Pool     *worker.Pool
	Netease  *netease.Client
	Telegram *telegram.Bot
	Build    BuildInfo
}

// BuildInfo provides build-time metadata.
type BuildInfo struct {
	RuntimeVer string
	BinVersion string
	CommitSHA  string
	BuildTime  string
	BuildArch  string
}

// New builds the application container.
func New(ctx context.Context, configPath string, build BuildInfo) (*App, error) {
	conf, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	log, err := logpkg.New(conf.GetString("LogLevel"))
	if err != nil {
		return nil, err
	}

	gormLogger := logpkg.NewGormLogger(log.Slog(), mapLogLevel(conf.GetString("LogLevel")))
	databasePath := conf.GetString("Database")
	if strings.TrimSpace(databasePath) == "" {
		databasePath = "cache.db"
	}

	repo, err := db.NewSQLiteRepository(databasePath, gormLogger)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	pool := worker.New(4)
	neteaseClient := netease.New(conf.GetString("MUSIC_U"), log)
	tele, err := telegram.New(conf, log)
	if err != nil {
		return nil, fmt.Errorf("init telegram: %w", err)
	}

	return &App{
		Config:   conf,
		Logger:   log,
		DB:       repo,
		Pool:     pool,
		Netease:  neteaseClient,
		Telegram: tele,
		Build:    build,
	}, nil
}

// Start initializes background services. Telegram startup is added in later waves.
func (a *App) Start(ctx context.Context) error {
	me, err := a.Telegram.GetMe(ctx)
	if err != nil {
		return err
	}
	botName := ""
	if me != nil {
		botName = me.Username
	}

	musicHandler := &handler.MusicHandler{
		Repo:            a.DB,
		Netease:         a.Netease,
		Pool:            a.Pool,
		Logger:          a.Logger,
		CacheDir:        "./cache",
		BotName:         botName,
		CheckMD5:        a.Config.GetBool("CheckMD5"),
		DownloadTimeout: time.Duration(a.Config.GetInt("DownloadTimeout")) * time.Second,
		ReverseProxy:    a.Config.GetString("ReverseProxy"),
	}

	router := &handler.Router{
		Music:     musicHandler,
		Search:    &handler.SearchHandler{Netease: a.Netease},
		Lyric:     &handler.LyricHandler{Netease: a.Netease, CacheDir: "./cache"},
		Recognize: &handler.RecognizeHandler{CacheDir: "./cache", Music: musicHandler},
		About:     &handler.AboutHandler{RuntimeVer: a.Build.RuntimeVer, BinVersion: a.Build.BinVersion, CommitSHA: a.Build.CommitSHA, BuildTime: a.Build.BuildTime, BuildArch: a.Build.BuildArch},
		Status:    &handler.StatusHandler{Repo: a.DB},
		RmCache:   &handler.RmCacheHandler{Repo: a.DB},
		Callback:  &handler.CallbackMusicHandler{Music: musicHandler, BotName: botName},
		Inline:    &handler.InlineSearchHandler{Repo: a.DB, Netease: a.Netease, BotName: botName},
	}

	router.Register(a.Telegram.Client(), botName)
	go a.Telegram.Start(ctx)
	return nil
}

// Shutdown releases resources.
func (a *App) Shutdown(ctx context.Context) error {
	if a.Pool != nil {
		return a.Pool.Shutdown(ctx)
	}
	return nil
}

func mapLogLevel(level string) gormlogger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "trace":
		return gormlogger.Info
	case "warn", "warning":
		return gormlogger.Warn
	case "error", "fatal", "panic":
		return gormlogger.Error
	case "info":
		fallthrough
	default:
		return gormlogger.Info
	}
}
