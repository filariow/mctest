package poll

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TODO: consider using https://github.com/kubernetes/apimachinery/tree/master/pkg/util/wait

var ErrPollerTimeout error = fmt.Errorf("poller timed out")

func Do(ctx context.Context, interval time.Duration, doFunc func(context.Context) error) error {
	df := func(ictx context.Context) (struct{}, error) {
		return struct{}{}, doFunc(ictx)
	}

	_, err := DoR(ctx, interval, df)
	return err
}

func DoR[R any](ctx context.Context, interval time.Duration, doFunc func(context.Context) (R, error)) (R, error) {
	errs := []error{}
	cr, ce := DoRC[R](ctx, interval, doFunc)
	for {
		select {
		case err, ok := <-ce:
			if !ok {
				var r R
				return r, errors.Join(errs...)
			}
			errs = append(errs, err)
		case r, ok := <-cr:
			if ok {
				return r, nil
			}
		}
	}
}

func DoRC[R any](ctx context.Context, interval time.Duration, doFunc func(context.Context) (R, error)) (<-chan R, <-chan error) {
	cr, ce := make(chan R), make(chan error)

	go func() {
		defer close(ce)
		defer close(cr)

		for {
			r, err := doFunc(ctx)

			if err != nil {
				ce <- err
				tr := time.NewTimer(interval)

				// wait for interval time to pass or context cancellation
				select {
				case <-ctx.Done():
					ce <- ErrPollerTimeout
					return
				case <-tr.C:
					continue
				}
			}

			cr <- r
			return
		}
	}()

	return cr, ce
}
