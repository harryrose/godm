package cache

import (
	"context"
	"sync"
	"time"
)

type TTL[K comparable, V any] struct {
	TTL                 time.Duration
	data                map[K]ttlItem[V]
	mux                 sync.RWMutex
	cleanLoopRunningMux sync.Mutex
}

func (t *TTL[K, V]) Get(k K) (out V, ok bool) {
	t.mux.RLock()
	defer t.mux.RUnlock()
	var tmp ttlItem[V]
	tmp, ok = t.data[k]
	if ok {
		if !tmp.isExpired() {
			out = tmp.value
		} else {
			// expired
			ok = false
		}
	}
	return
}

func (t *TTL[K, V]) Set(k K, v V) {
	t.mux.Lock()
	defer t.mux.Unlock()

	if t.data == nil {
		t.data = make(map[K]ttlItem[V])
	}

	t.data[k] = ttlItem[V]{
		exp:   time.Now().Add(t.TTL),
		value: v,
	}
}

func (t *TTL[K, V]) Clean() {
	t.mux.Lock()
	defer t.mux.Unlock()

	for k, v := range t.data {
		if v.isExpired() {
			delete(t.data, k)
		}
	}
}

func (t *TTL[K, V]) CleanLoopAsync(ctx context.Context, period time.Duration) {
	ticker := time.Tick(period)
	go func() {
		if got := t.cleanLoopRunningMux.TryLock(); !got {
			// already running
			return
		}
		defer t.cleanLoopRunningMux.Unlock()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker:
				t.Clean()
			}
		}
	}()
}

type ttlItem[V any] struct {
	exp   time.Time
	value V
}

func (t *ttlItem[V]) isExpired() bool {
	if t.exp.After(time.Now()) {
		return false
	}
	return true
}
