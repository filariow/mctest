package testrun

import (
	"context"
	"fmt"

	econtext "github.com/filariow/mctest/pkg/context"
)

const (
	keyTestFolder               string = "test-folder"
	keyTimeoutContextCancelFunc string = "context-cancel"
)

var (
	ErrTestFolderNotFound error = fmt.Errorf("test folder not found in context")
)

// timeout
func TimeoutContextCancelIntoContext(ctx context.Context, value context.CancelFunc) context.Context {
	return econtext.IntoContext(ctx, keyTimeoutContextCancelFunc, value)
}

func TimeoutContextCancelFromContext(ctx context.Context) (*context.CancelFunc, error) {
	return econtext.FromContext[context.CancelFunc](ctx, keyTimeoutContextCancelFunc)
}

func TimeoutContextCancelFromContextOrDie(ctx context.Context) context.CancelFunc {
	return econtext.FromContextOrDie[context.CancelFunc](ctx, keyTimeoutContextCancelFunc)
}

// test folder
func TestFolderIntoContext(ctx context.Context, value string) context.Context {
	return econtext.IntoContext(ctx, keyTestFolder, value)
}

func TestFolderFromContext(ctx context.Context) (*string, error) {
	return econtext.FromContext[string](ctx, keyTestFolder)
}

func TestFolderFromContextOrDie(ctx context.Context) string {
	return econtext.FromContextOrDie[string](ctx, keyTestFolder)
}
