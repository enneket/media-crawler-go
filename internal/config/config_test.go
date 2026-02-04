package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfig_SaveDataOptionExcelAlias(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	dir := t.TempDir()
	cfg := []byte("SAVE_DATA_OPTION: \"excel\"\nSTORE_BACKEND: \"MongoDB\"\nCRAWLER_TYPE: \"SEARCH\"\nLOGIN_TYPE: \"COOKIE\"\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), cfg, 0644); err != nil {
		t.Fatal(err)
	}

	if err := LoadConfig(dir); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if AppConfig.SaveDataOption != "xlsx" {
		t.Fatalf("SaveDataOption = %q, want %q", AppConfig.SaveDataOption, "xlsx")
	}
	if AppConfig.StoreBackend != "mongodb" {
		t.Fatalf("StoreBackend = %q, want %q", AppConfig.StoreBackend, "mongodb")
	}
	if AppConfig.CrawlerType != "search" {
		t.Fatalf("CrawlerType = %q, want %q", AppConfig.CrawlerType, "search")
	}
	if AppConfig.LoginType != "cookie" {
		t.Fatalf("LoginType = %q, want %q", AppConfig.LoginType, "cookie")
	}
}

