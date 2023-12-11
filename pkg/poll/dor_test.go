package poll_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/filariow/mctest/pkg/poll"
)

func Test_DoR(t *testing.T) {
	t.Run("immediate return on no error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		counter, expected := 0, 1
		doFunc := func(ctx context.Context) (interface{}, error) {
			counter++
			return nil, nil
		}

		c := make(chan struct{}, 1)
		go func() {
			_, err := poll.DoR(ctx, 50*time.Millisecond, doFunc)
			if err != nil {
				t.Error(err)
			}
			c <- struct{}{}
		}()

		tr := time.NewTimer(100 * time.Millisecond)
		select {
		case <-tr.C:
			t.Error("taking too much")
		case <-c:
			if counter != expected {
				t.Errorf("expected counter to be %d, got %d", expected, counter)
			}
		}
	})

	t.Run("retry until deadline", func(t *testing.T) {
		t.Parallel()

		it, wt := 100*time.Millisecond, 350*time.Millisecond
		expected := int(wt/it) + 1
		errs := []error{}
		doFunc := func(ctx context.Context) (interface{}, error) {
			err := fmt.Errorf("dummy")
			errs = append(errs, err)
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), wt)
		defer cancel()

		_, err := poll.DoR(ctx, it, doFunc)
		if !errors.Is(err, poll.ErrPollerTimeout) {
			t.Errorf("expected ErrPollerTimeout (%v), got %v. Counter is %d", poll.ErrPollerTimeout, err, len(errs))
		}

		if counter := len(errs); counter != expected {
			t.Errorf("expected counter to be %d, got %d", expected, counter)
		}
	})

	t.Run("retry until success", func(t *testing.T) {
		t.Parallel()

		counter, expected := 0, 2
		doFunc := func(ctx context.Context) (interface{}, error) {
			if counter != 2 {
				counter++
				return nil, fmt.Errorf("dummy")
			}

			return nil, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_, err := poll.DoR(ctx, 20*time.Millisecond, doFunc)
		if errors.Is(err, poll.ErrPollerTimeout) {
			t.Error(err)
		}

		if counter != expected {
			t.Errorf("expected counter to be %d, got %d", expected, counter)
		}
	})
}
