package asyncx

import "context"

func AsyncAll[T any, R any](ctx context.Context, items []T, fn func(ctx context.Context, item T) (R, error)) ([]R, error) {
	results := make(chan R, len(items))
	errs := make(chan error, 1)

	for _, item := range items {
		go func(i T) {
			result, err := fn(ctx, i)
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}(item)
	}

	var collected []R
	for i := 0; i < len(items); i++ {
		select {
		case err := <-errs:
			return nil, err
		case result := <-results:
			collected = append(collected, result)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return collected, nil
}
