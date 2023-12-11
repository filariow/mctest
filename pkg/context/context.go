package context

import (
	"context"
	"fmt"
)

var (
	ErrKeyNotFound error = fmt.Errorf("error key not found in context")
)

// general
func IntoContext[T any](ctx context.Context, key string, value T) context.Context {
	return context.WithValue(ctx, key, value)
}

func FromContext[T any](ctx context.Context, key string) (*T, error) {
	v, ok := ctx.Value(key).(T)
	if !ok {
		return nil, fmt.Errorf("%w: key=%v", ErrKeyNotFound, key)
	}
	return &v, nil
}

func FromContextOrDie[T any](ctx context.Context, key string) T {
	v, err := FromContext[T](ctx, key)
	if err != nil {
		panic(err)
	}
	return *v
}
