package downloader

import (
	"context"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"sync"
)

const (
	eventTransferSizeBytes = 128 * 1024
)

type RateLimiter struct {
	bytesPerSecond int
	limMux         sync.RWMutex
	lim            *rate.Limiter
}

func NewRateLimiter(bytesPerSecond int) *RateLimiter {
	r := &RateLimiter{}
	r.SetRateLimit(bytesPerSecond)
	return r
}

func (r *RateLimiter) limiter() *rate.Limiter {
	r.limMux.RLock()
	defer r.limMux.RUnlock()
	tmp := r.lim
	return tmp
}

func (r *RateLimiter) setLimiter(l *rate.Limiter) {
	r.limMux.Lock()
	defer r.limMux.Unlock()
	r.lim = l
}

// SetRateLimit sets the maximum transfer rate that the rate limiter will allow.
// If bytesPerSecond is zero or negative, no limit is applied.
// Otherwise, the rate limit is set to the maximum of bytesPerSecond and eventTransferSizeBytes
func (r *RateLimiter) SetRateLimit(bytesPerSecond int) {
	switch {
	case bytesPerSecond <= 0:
		r.setLimiter(rate.NewLimiter(rate.Inf, 1))

	case bytesPerSecond < eventTransferSizeBytes:
		bytesPerSecond = eventTransferSizeBytes
		fallthrough

	default:
		l := rate.NewLimiter(rate.Limit(bytesPerSecond), bytesPerSecond)
		r.setLimiter(l)
	}
	r.bytesPerSecond = bytesPerSecond
}

// Transfer transfers from the reader to the writer at a specified maximum
// transfer rate.  If there is an error reading or writing, an error is returned.
// Transfer may be called multiple times in parallel, in which case the transfer
// rate is shared across all parallel invocations of Transfer.
func (r *RateLimiter) Transfer(ctx context.Context, rd io.Reader, w io.Writer) error {
	buf := make([]byte, eventTransferSizeBytes)

	for {
		if err := r.limiter().WaitN(ctx, eventTransferSizeBytes); err != nil {
			return err
		}

		n, rdErr := rd.Read(buf)
		_, wErr := w.Write(buf[:n])

		if rdErr != nil {
			if rdErr == io.EOF {
				return nil
			}
			return fmt.Errorf("read error: %w", rdErr)
		}
		if wErr != nil {
			return fmt.Errorf("write error: %w", rdErr)
		}
	}
}
