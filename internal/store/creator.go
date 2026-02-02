package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func CreatorDir(secUserID string) string {
	return filepath.Join(PlatformDir(), "creators", secUserID)
}

func SaveCreatorProfile(secUserID string, profile any) error {
	if err := sqliteUpsertCreator(secUserID, profile); err != nil {
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
	return os.WriteFile(path, b, 0644)
}
