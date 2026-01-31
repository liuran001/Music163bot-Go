package config

import (
	"path/filepath"
	"testing"
)

func TestLoadINI(t *testing.T) {
	path := filepath.Join("..", "..", "config_example.ini")
	conf, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if conf.GetString("BOT_TOKEN") == "" {
		t.Fatalf("expected BOT_TOKEN to be present")
	}

	admins := conf.GetIntSlice("BotAdmin")
	if len(admins) == 0 {
		t.Fatalf("expected BotAdmin to be parsed")
	}
}
