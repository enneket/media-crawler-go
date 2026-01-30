package douyin

import "testing"

func TestPickAwemeInfoFromSearchItem(t *testing.T) {
	item1 := map[string]any{
		"aweme_info": map[string]any{"aweme_id": "1"},
	}
	got := pickAwemeInfoFromSearchItem(item1)
	if got == nil || got["aweme_id"] != "1" {
		t.Fatalf("expected aweme_info")
	}

	item2 := map[string]any{
		"aweme_mix_info": map[string]any{
			"mix_items": []any{map[string]any{"aweme_id": "2"}},
		},
	}
	got = pickAwemeInfoFromSearchItem(item2)
	if got == nil || got["aweme_id"] != "2" {
		t.Fatalf("expected aweme_mix_info.mix_items[0]")
	}
}
