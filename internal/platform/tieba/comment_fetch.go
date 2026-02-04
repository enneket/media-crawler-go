package tieba

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func threadPageURL(threadID string, page int) string {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return ""
	}
	if page <= 0 {
		page = 1
	}
	u := fmt.Sprintf("https://tieba.baidu.com/p/%s", url.PathEscape(threadID))
	if page == 1 {
		return u
	}
	return u + "?pn=" + strconv.Itoa(page)
}

func subCommentPageURL(threadID string, parentPostID string, forumID string, page int) string {
	threadID = strings.TrimSpace(threadID)
	parentPostID = strings.TrimSpace(parentPostID)
	forumID = strings.TrimSpace(forumID)
	if threadID == "" || parentPostID == "" || forumID == "" {
		return ""
	}
	if page <= 0 {
		page = 1
	}
	return fmt.Sprintf(
		"https://tieba.baidu.com/p/comment?tid=%s&pid=%s&fid=%s&pn=%d",
		url.QueryEscape(threadID),
		url.QueryEscape(parentPostID),
		url.QueryEscape(forumID),
		page,
	)
}

func fetchAllThreadComments(ctx context.Context, client fetchClient, threadID string, noteID string, max int, sleepSec int, enableSub bool) ([]Comment, error) {
	threadID = strings.TrimSpace(threadID)
	noteID = strings.TrimSpace(noteID)
	if client == nil || threadID == "" || noteID == "" || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}
	if sleepSec < 0 {
		sleepSec = 0
	}

	out := make([]Comment, 0, 128)
	seen := map[string]struct{}{}

	for page := 1; len(out) < max; page++ {
		u := threadPageURL(threadID, page)
		if u == "" {
			break
		}
		res, err := client.FetchHTML(ctx, u)
		if err != nil {
			return out, err
		}
		parents := parseParentCommentsFromHTML(res.Body, noteID)
		if len(parents) == 0 {
			break
		}
		addedThisPage := 0
		for _, p := range parents {
			c := p.Comment
			if c.CommentID == "" {
				continue
			}
			if _, ok := seen[c.CommentID]; ok {
				continue
			}
			seen[c.CommentID] = struct{}{}
			out = append(out, c)
			addedThisPage++
			if len(out) >= max {
				break
			}

			if enableSub && p.SubCount > 0 {
				sub, err := fetchAllSubComments(ctx, client, threadID, noteID, c.CommentID, p.ForumID, max-len(out), sleepSec)
				if err != nil {
					return out, err
				}
				for _, sc := range sub {
					if sc.CommentID == "" {
						continue
					}
					if _, ok := seen[sc.CommentID]; ok {
						continue
					}
					seen[sc.CommentID] = struct{}{}
					out = append(out, sc)
					if len(out) >= max {
						break
					}
				}
			}
			if len(out) >= max {
				break
			}
		}
		if addedThisPage == 0 {
			break
		}
		if sleepSec > 0 {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(time.Duration(sleepSec) * time.Second):
			}
		}
	}
	return out, nil
}

func fetchAllSubComments(ctx context.Context, client fetchClient, threadID string, noteID string, parentPostID string, forumID string, max int, sleepSec int) ([]Comment, error) {
	if client == nil || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}
	if sleepSec < 0 {
		sleepSec = 0
	}
	out := make([]Comment, 0, 64)
	seen := map[string]struct{}{}
	for page := 1; len(out) < max; page++ {
		u := subCommentPageURL(threadID, parentPostID, forumID, page)
		if u == "" {
			break
		}
		res, err := client.FetchHTML(ctx, u)
		if err != nil {
			return out, err
		}
		subs := parseSubCommentsFromHTML(res.Body, noteID, parentPostID)
		if len(subs) == 0 {
			break
		}
		added := 0
		for _, sm := range subs {
			c := sm.Comment
			if c.CommentID == "" {
				continue
			}
			if _, ok := seen[c.CommentID]; ok {
				continue
			}
			seen[c.CommentID] = struct{}{}
			out = append(out, c)
			added++
			if len(out) >= max {
				break
			}
		}
		if added == 0 {
			break
		}
		if sleepSec > 0 {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(time.Duration(sleepSec) * time.Second):
			}
		}
	}
	return out, nil
}

