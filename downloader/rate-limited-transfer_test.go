package downloader

import (
	"bytes"
	"context"
	"crypto/rand"
	"reflect"
	"testing"
	"time"
)

func TestRateLimiter_Transfer(t *testing.T) {
	const (
		bytesPerSecond              = 200 * 1024
		expectedTransferTimeSeconds = 10
		inputSizeBytes              = bytesPerSecond * expectedTransferTimeSeconds
		tolerance                   = time.Second
	)

	testInput := make([]byte, inputSizeBytes)

	l, err := rand.Reader.Read(testInput)
	if err != nil {
		t.Fatalf("unable to build test input: %v", err)
	}
	if expectedLen := len(testInput); l != expectedLen {
		t.Fatalf("unexpected read length while building input: expectd %v bytes but read %v bytes", expectedLen, l)
	}

	w := &mockWriter{}

	uut := NewRateLimiter(bytesPerSecond)
	start := time.Now()
	if err := uut.Transfer(context.Background(), bytes.NewReader(testInput), w); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	duration := time.Since(start)

	difference := (duration - expectedTransferTimeSeconds*time.Second).Abs()

	if difference > tolerance {
		t.Errorf("transfer time was expected to be %v seconds ± %v, but was %v", expectedTransferTimeSeconds, tolerance, duration)
	}

	if !reflect.DeepEqual(w.buf, testInput) {
		t.Errorf("transferred data did not match input")
	}
}

type mockWriter struct {
	buf []byte
}

func (w *mockWriter) Write(b []byte) (int, error) {
	w.buf = append(w.buf, b...)
	return len(b), nil
}
