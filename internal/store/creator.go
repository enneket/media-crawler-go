package store

import (
	"encoding/json"
	"media-crawler-go/internal/config"
	"os"
	"path/filepath"
)

func CreatorDir(secUserID string) string {
	return filepath.Join(PlatformDir(), "creators", secUserID)
}

func SaveCreatorProfile(secUserID string, profile any) error {
	if err := sqlUpsertCreator(secUserID, profile); err != nil {
		return err
	}
	dir := CreatorDir(secUserID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "profile.json")
	b, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return err
	}
	_ = pythonCompatAppendJSON("creators", profile)

	if config.AppConfig.SaveDataOption == "xlsx_book" || config.AppConfig.SaveDataOption == "excel" {
		_ = AppendBookCreator(secUserID, profile)
	}

	return nil
}

func SaveCreatorDynamics(secUserID string, dynamics any) error {
	dir := CreatorDir(secUserID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "dynamics.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	items, ok := dynamics.([]any)
	if !ok {
		items = []any{dynamics}
	}

	encoder := json.NewEncoder(f)
	for _, item := range items {
		if err := encoder.Encode(item); err != nil {
			return err
		}
	}
	return nil
}
