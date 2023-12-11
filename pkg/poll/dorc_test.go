package poll_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/filariow/mctest/pkg/poll"
)

func Test_DoRC(t *testing.T) {
	t.Run("immediate return on no error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		er := "success"
		doFunc := func(ctx context.Context) (*string, error) {
			return &er, nil
		}

		tr := time.NewTimer(100 * time.Millisecond)
		cr, ce := poll.DoRC(ctx, 50*time.Millisecond, doFunc)
		for {
			select {
			case r, ok := <-cr:
				if !ok {
					t.Fatal("expected channel to be open")
				}

				if &er != r {
					t.Fatalf("expected result %v, obtained %v", er, r)
				}

				select {
				case <-cr:
				default:
					t.Fatalf("result channel expected to be closed now, found open")
				}

				select {
				case <-ce:
				default:
					t.Fatalf("errors channel expected to be closed now, found open")
				}

				return
			case err := <-ce:
				t.Fatalf("unexpected error: %v", err)
			case <-tr.C:
				t.Fatalf("taking too much")
			}
		}

	})

	t.Run("retry until deadline", func(t *testing.T) {
		t.Parallel()

		errs := []error{}
		it, wt := 100*time.Millisecond, 350*time.Millisecond
		expected := int(wt/it) + 2
		doFunc := func(ctx context.Context) (interface{}, error) {
			return nil, fmt.Errorf("dummy")
		}

		ctx, cancel := context.WithTimeout(context.Background(), wt)
		defer cancel()

		tctx, tcancel := context.WithTimeout(context.Background(), wt*2)
		defer tcancel()

		_, ce := poll.DoRC(ctx, it, doFunc)
		for {
			select {
			case err, ok := <-ce:
				if !ok {
					if counter := len(errs); counter != expected {
						t.Fatalf("expected %d errors, got %d", expected, counter)
					}

					if ej := errors.Join(errs...); errors.Is(poll.ErrPollerTimeout, ej) {
						t.Fatalf("expected ErrPollerTimeout (%v), got %v", poll.ErrPollerTimeout, ej)
					}
					return
				}
				errs = append(errs, err)
			case <-tctx.Done():
				t.Fatalf("test timed out")
			}
		}
	})

	t.Run("retry until success", func(t *testing.T) {
		t.Parallel()

		expected := 2
		er := "success"
		eerrs, oerrs := []error{}, []error{}
		doFunc := func(ctx context.Context) (*string, error) {
			if counter := len(eerrs); counter != 2 {
				err := fmt.Errorf("dummy")
				eerrs = append(eerrs, err)
				return nil, err
			}

			return &er, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		cr, ce := poll.DoRC(ctx, 20*time.Millisecond, doFunc)
		for {
			select {
			case <-ctx.Done():
				t.Fatalf("test timed out")
			case err, ok := <-ce:
				if ok {
					oerrs = append(oerrs, err)
				}
			case r, ok := <-cr:
				if !ok {
					if counter := len(oerrs); counter != expected {
						t.Errorf("expected counter to be %d, got %d", expected, counter)
					}
					return
				}

				if &er != r {
					t.Errorf("expected result to be %s, got %s", er, *r)
				}
			}
		}

	})
}
