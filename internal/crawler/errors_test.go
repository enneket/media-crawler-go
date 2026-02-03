package crawler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestKindOf(t *testing.T) {
	{
		err := context.Canceled
		if got := KindOf(err); got != ErrorKindCanceled {
			t.Fatalf("canceled got=%s", got)
		}
	}
	{
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		<-ctx.Done()
		if got := KindOf(ctx.Err()); got != ErrorKindTimeout {
			t.Fatalf("deadline got=%s", got)
		}
	}
	{
		err := Error{Kind: ErrorKindInvalidInput, Platform: "xhs", Msg: "bad"}
		if got := KindOf(err); got != ErrorKindInvalidInput {
			t.Fatalf("custom kind got=%s", got)
		}
	}
	{
		err := errors.New("http status=429 body=xxx")
		if got := KindOf(err); got != ErrorKindRateLimited {
			t.Fatalf("429 got=%s", got)
		}
	}
	{
		err := errors.New("http status=403 body=xxx")
		if got := KindOf(err); got != ErrorKindForbidden {
			t.Fatalf("403 got=%s", got)
		}
	}
	{
		err := errors.New("http status=500 body=xxx")
		if got := KindOf(err); got != ErrorKindHTTP {
			t.Fatalf("500 got=%s", got)
		}
	}
	{
		err := NewHTTPStatusError("x", "u", 429, "nope")
		if got := KindOf(err); got != ErrorKindRateLimited {
			t.Fatalf("wrapped 429 got=%s", got)
		}
	}
	{
		err := errors.New("something else")
		if got := KindOf(err); got != ErrorKindUnknown {
			t.Fatalf("unknown got=%s", got)
		}
	}
}
