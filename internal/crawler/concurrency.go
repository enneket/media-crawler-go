package crawler

import (
	"context"
)

type ItemResult struct {
	Processed int
	Succeeded int
	Failed    int
}

func ForEachLimit[T any](ctx context.Context, items []T, limit int, fn func(context.Context, T) error) ItemResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if limit <= 1 {
		var out ItemResult
		for _, it := range items {
			select {
			case <-ctx.Done():
				return out
			default:
			}
			out.Processed++
			if err := fn(ctx, it); err != nil {
				out.Failed++
				continue
			}
			out.Succeeded++
		}
		return out
	}

	jobs := make(chan T)
	res := make(chan error, limit)

	for i := 0; i < limit; i++ {
		go func() {
			for it := range jobs {
				res <- fn(ctx, it)
			}
		}()
	}

	var out ItemResult
	stopped := false
	for _, it := range items {
		if stopped {
			break
		}
		select {
		case <-ctx.Done():
			stopped = true
		case jobs <- it:
			out.Processed++
		}
	}
	close(jobs)

	for i := 0; i < out.Processed; i++ {
		err := <-res
		if err != nil {
			out.Failed++
			continue
		}
		out.Succeeded++
	}
	return out
}
