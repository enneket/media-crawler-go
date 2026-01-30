package xhs

import "testing"

func TestExtractCreatorUserPageData(t *testing.T) {
	html := `
<html><head></head><body>
<script>window.__INITIAL_STATE__={"user":{"userPageData":{"basicInfo":{"nickname":"nick","gender":1,"images":["a"],"desc":"hi {brace}","ipLocation":"北京"},"interactions":[{"type":"follows","count":2},{"type":"fans","count":3},{"type":"interaction","count":9}],"tags":[{"tagType":"interest","name":"Go"}]}}}</script>
</body></html>`

	upd, err := ExtractCreatorUserPageData(html)
	if err != nil {
		t.Fatalf("ExtractCreatorUserPageData err: %v", err)
	}
	rec, err := BuildCreatorRecord("u1", upd)
	if err != nil {
		t.Fatalf("BuildCreatorRecord err: %v", err)
	}
	if rec.UserID != "u1" || rec.Nickname != "nick" || rec.Gender != "Female" {
		t.Fatalf("unexpected record: %+v", rec)
	}
	if rec.Follows != 2 || rec.Fans != 3 || rec.Interaction != 9 {
		t.Fatalf("unexpected interaction counts: %+v", rec)
	}
	if rec.TagList == "" {
		t.Fatalf("expected tag_list not empty")
	}
}

func TestExtractInitialStateUndefined(t *testing.T) {
	html := `<script>window.__INITIAL_STATE__={"user":{"userPageData":{"basicInfo":{"nickname":"n","foo":undefined}}}}</script>`
	_, err := ExtractCreatorUserPageData(html)
	if err != nil {
		t.Fatalf("expected undefined normalization to work, got err: %v", err)
	}
}
