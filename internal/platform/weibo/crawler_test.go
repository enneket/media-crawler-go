package weibo

import (
	"context"
	"encoding/json"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"os"
	"path/filepath"
	"testing"
)

type fakeClientWithComments struct{}

func (f fakeClientWithComments) Show(ctx context.Context, id string) (ShowResponse, error) {
	b, _ := json.Marshal(map[string]any{"id": id})
	return ShowResponse{Ok: 1, Data: b}, nil
}

func (f fakeClientWithComments) SearchByKeyword(ctx context.Context, keyword string, page int, searchType string) (GetIndexResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return GetIndexResponse{Ok: 1, Data: b}, nil
}

func (f fakeClientWithComments) CreatorInfo(ctx context.Context, creatorID string) (GetIndexResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return GetIndexResponse{Ok: 1, Data: b}, nil
}

func (f fakeClientWithComments) NotesByCreator(ctx context.Context, creatorID string, containerID string, sinceID string) (GetIndexResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return GetIndexResponse{Ok: 1, Data: b}, nil
}

func (f fakeClientWithComments) GetNoteComments(ctx context.Context, noteID string, maxID int64, maxIDType int) (hotflowData, error) {
	return hotflowData{
		MaxID:     0,
		MaxIDType: 0,
		Data: []hotflowComment{
			{
				ID:        "c1",
				RootID:    "",
				Text:      "<span>hello</span>",
				CreatedAt: "Mon Jan 02 15:04:05 +0800 2006",
				LikeCount: 5,
				User: hotflowUser{
					ID:         "u1",
					ScreenName: "nick",
				},
				Comments: []hotflowComment{
					{
						ID:        "c2",
						RootID:    "c1",
						Text:      "sub",
						CreatedAt: "Mon Jan 02 15:04:06 +0800 2006",
						LikeCount: 1,
						User: hotflowUser{
							ID:         "u2",
							ScreenName: "nick2",
						},
					},
				},
			},
		},
	}, nil
}

func TestCrawlerCommentsSaved(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	config.AppConfig = config.Config{
		Platform:             "weibo",
		StoreBackend:         "file",
		SaveDataOption:       "json",
		DataDir:              "data",
		EnableGetComments:    true,
		EnableGetSubComments: true,
		CrawlerMaxComments:   10,
		CrawlerMaxSleepSec:   0,
	}

	c := NewCrawlerWithClient(fakeClientWithComments{})
	req := crawler.Request{Platform: "weibo", Mode: crawler.ModeDetail, Inputs: []string{"4KjD8oZ4D"}, Concurrency: 1}
	if _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}

	note := filepath.Join("data", "weibo", "notes", "4KjD8oZ4D", "note.json")
	if _, err := os.Stat(note); err != nil {
		t.Fatalf("expected note saved at %s: %v", note, err)
	}
	noteComments := filepath.Join("data", "weibo", "notes", "4KjD8oZ4D", "comments.jsonl")
	if _, err := os.Stat(noteComments); err != nil {
		t.Fatalf("expected comments saved at %s: %v", noteComments, err)
	}
	globalComments := filepath.Join("data", "weibo", "comments.jsonl")
	if _, err := os.Stat(globalComments); err != nil {
		t.Fatalf("expected global comments saved at %s: %v", globalComments, err)
	}
}

