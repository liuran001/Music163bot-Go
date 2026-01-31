package config

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

// Config wraps viper and provides typed accessors.
type Config struct {
	v        *viper.Viper
	botAdmin []int
}

// Load reads an INI config file and prepares defaults.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("MUSIC163BOT")
	v.AutomaticEnv()

	setDefaults(v)

	if strings.EqualFold(filepath.Ext(path), ".ini") {
		if err := loadINI(v, path); err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	} else {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	return &Config{
		v:        v,
		botAdmin: parseIntList(v.GetString("BotAdmin")),
	}, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("BotAPI", "https://api.telegram.org")
	v.SetDefault("BotDebug", false)
	v.SetDefault("DownloadTimeout", 60)
	v.SetDefault("CheckMD5", true)
	v.SetDefault("Database", "cache.db")
	v.SetDefault("LogLevel", "info")
}

// GetString returns a string value.
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt returns an int value.
func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetBool returns a bool value.
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// GetIntSlice returns a slice of ints. BotAdmin is parsed from comma-separated values.
func (c *Config) GetIntSlice(key string) []int {
	if key == "BotAdmin" {
		return append([]int(nil), c.botAdmin...)
	}
	return c.v.GetIntSlice(key)
}

func parseIntList(value string) []int {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		num, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		result = append(result, num)
	}
	return result
}

func loadINI(v *viper.Viper, path string) error {
	cfg, err := ini.Load(path)
	if err != nil {
		return err
	}

	for _, key := range cfg.Section("").Keys() {
		v.Set(key.Name(), key.Value())
	}
	return nil
}
