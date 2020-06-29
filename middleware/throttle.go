package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

const (
	errCapacityExceeded = "Server capacity exceeded."
	errTimedOut         = "Timed out while waiting for a pending request to complete."
	errContextCanceled  = "Context was canceled."
)

var (
	defaultBacklogTimeout = time.Second * 60
)

// ThrottleOpts represents a set of throttling options.
type ThrottleOpts struct {
	Limit          int
	BacklogLimit   int
	BacklogTimeout time.Duration
	RetryAfterFn   func(ctxDone bool) time.Duration
}

// Throttle is a middleware that limits number of currently processed requests
// at a time across all users. Note: Throttle is not a rate-limiter per user,
// instead it just puts a ceiling on the number of currentl in-flight requests
// being processed from the point from where the Throttle middleware is mounted.
func Throttle(limit int) func(fchi.Handler) fchi.Handler {
	return ThrottleWithOpts(ThrottleOpts{Limit: limit, BacklogTimeout: defaultBacklogTimeout})
}

// ThrottleBacklog is a middleware that limits number of currently processed
// requests at a time and provides a backlog for holding a finite number of
// pending requests.
func ThrottleBacklog(limit int, backlogLimit int, backlogTimeout time.Duration) func(fchi.Handler) fchi.Handler {
	return ThrottleWithOpts(ThrottleOpts{Limit: limit, BacklogLimit: backlogLimit, BacklogTimeout: backlogTimeout})
}

// ThrottleWithOpts is a middleware that limits number of currently processed requests using passed ThrottleOpts.
func ThrottleWithOpts(opts ThrottleOpts) func(fchi.Handler) fchi.Handler {
	if opts.Limit < 1 {
		panic("chi/middleware: Throttle expects limit > 0")
	}

	if opts.BacklogLimit < 0 {
		panic("chi/middleware: Throttle expects backlogLimit to be positive")
	}

	t := throttler{
		tokens:         make(chan token, opts.Limit),
		backlogTokens:  make(chan token, opts.Limit+opts.BacklogLimit),
		backlogTimeout: opts.BacklogTimeout,
		retryAfterFn:   opts.RetryAfterFn,
	}

	// Filling tokens.
	for i := 0; i < opts.Limit+opts.BacklogLimit; i++ {
		if i < opts.Limit {
			t.tokens <- token{}
		}
		t.backlogTokens <- token{}
	}

	return func(next fchi.Handler) fchi.Handler {
		fn := func(ctx context.Context, rc *fasthttp.RequestCtx) {
			select {

			case <-ctx.Done():
				t.setRetryAfterHeaderIfNeeded(rc, true)
				rc.Error(errContextCanceled, fasthttp.StatusServiceUnavailable)
				return

			case btok := <-t.backlogTokens:
				timer := time.NewTimer(t.backlogTimeout)

				defer func() {
					t.backlogTokens <- btok
				}()

				select {
				case <-timer.C:
					t.setRetryAfterHeaderIfNeeded(rc, false)
					rc.Error(errTimedOut, fasthttp.StatusServiceUnavailable)
					return
				case <-ctx.Done():
					timer.Stop()
					t.setRetryAfterHeaderIfNeeded(rc, true)
					rc.Error(errContextCanceled, fasthttp.StatusServiceUnavailable)
					return
				case tok := <-t.tokens:
					defer func() {
						timer.Stop()
						t.tokens <- tok
					}()
					next.ServeHTTP(ctx, rc)
				}
				return

			default:
				t.setRetryAfterHeaderIfNeeded(rc, false)
				rc.Error(errCapacityExceeded, fasthttp.StatusServiceUnavailable)
				return
			}
		}

		return fchi.HandlerFunc(fn)
	}
}

// token represents a request that is being processed.
type token struct{}

// throttler limits number of currently processed requests at a time.
type throttler struct {
	tokens         chan token
	backlogTokens  chan token
	backlogTimeout time.Duration
	retryAfterFn   func(ctxDone bool) time.Duration
}

// setRetryAfterHeaderIfNeeded sets Retry-After HTTP header if corresponding retryAfterFn option of throttler is initialized.
func (t throttler) setRetryAfterHeaderIfNeeded(rc *fasthttp.RequestCtx, ctxDone bool) {
	if t.retryAfterFn == nil {
		return
	}
	rc.Response.Header.Set("Retry-After", strconv.Itoa(int(t.retryAfterFn(ctxDone).Seconds())))
}
