package downloader

import (
	"context"
	"fmt"
	"github.com/harryrose/godm/downloader/queue"
	"github.com/harryrose/godm/downloader/reader"
	"github.com/harryrose/godm/downloader/writer"
	"io"
	"log"
	"sync/atomic"
	"time"
)

const (
	updatePeriod = 3 * time.Second
)

func Run(ctx context.Context, client queue.QueueServiceClient, pollPeriod time.Duration, queueName string, rateLimitBytesPerSecond int) {
	ticker := time.Tick(pollPeriod)
	for range ticker {
		claimed, err := client.ClaimNextItem(ctx, &queue.ClaimNextItemInput{
			Queue: &queue.Identifier{
				Id: queueName,
			},
		})

		if err != nil {

			continue
		}
		if claimed.Id == nil {
			continue
		}

		src := claimed.Item.Source.Url
		dst := claimed.Item.Destination.Url
		id := claimed.Id.Id

		bytesWritten, totalSizeBytes, err := handleItem(ctx, client, id, src, dst, rateLimitBytesPerSecond)
		if err != nil {
			log.Printf("download failed: %v\n", err)
			_, err := client.SetItemState(ctx, &queue.SetItemStateInput{
				Item: &queue.Identifier{Id: id},
				State: &queue.ItemState{
					State:           queue.ItemState_ITEM_STATE_FAILED,
					TotalSizeBytes:  uint64(totalSizeBytes),
					DownloadedBytes: uint64(bytesWritten),
					Message:         err.Error(),
				},
			})
			if err != nil {
				log.Printf("error setting item state to failed: %v", err)
			}
		} else {
			_, err := client.SetItemState(ctx, &queue.SetItemStateInput{
				Item: &queue.Identifier{Id: id},
				State: &queue.ItemState{
					State:           queue.ItemState_ITEM_STATE_COMPLETE,
					TotalSizeBytes:  uint64(totalSizeBytes),
					DownloadedBytes: uint64(bytesWritten),
				},
			})
			if err != nil {
				log.Printf("error setting item state to complete: %v", err)
			}
		}
	}
}

func handleItem(ctx context.Context, client queue.QueueServiceClient, id, src, dst string, rateLimitBytesPerSecond int) (int64, int64, error) {
	rdr, err := reader.BuildFromURL(src)
	if err != nil {
		return 0, 0, fmt.Errorf("error constructing downloader for url %v: %w", src, err)
	}
	wrt, err := writer.BuildFromURL(dst)
	if err != nil {
		return 0, 0, fmt.Errorf("error constructing writer for url %v: %w", dst, err)
	}
	r, totalSizeBytes, err := rdr.OpenReadCloser()
	if err != nil {
		return 0, totalSizeBytes, fmt.Errorf("error opening %v: %w", src, err)
	}
	defer r.Close()

	w, err := wrt.OpenWriteCloser()
	if err != nil {
		return 0, totalSizeBytes, fmt.Errorf("error opening %v: %w", dst, err)
	}
	defer w.Close()

	transferer := NewRateLimiter(rateLimitBytesPerSecond)

	cw := &AsyncByteCountingWriter{W: w}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		tick := time.Tick(updatePeriod)
		for {
			select {
			case <-ctx.Done():
				return

			case <-tick:
				bytesWritten := cw.BytesWritten()
				log.Printf("downloading item %v -- %v of %v bytes\n", id, bytesWritten, totalSizeBytes)
				_, err := client.SetItemState(ctx, &queue.SetItemStateInput{
					Item: &queue.Identifier{Id: id},
					State: &queue.ItemState{
						State:           queue.ItemState_ITEM_STATE_DOWNLOADING,
						TotalSizeBytes:  uint64(totalSizeBytes),
						DownloadedBytes: uint64(bytesWritten),
						Message:         "",
					},
				})
				if err != nil {
					log.Printf("error updating state for item %v: %v\n", id, err)
				}
			}
		}
	}()

	if err := transferer.Transfer(ctx, r, cw); err != nil {
		return cw.BytesWritten(), totalSizeBytes, fmt.Errorf("transfer error: %w", err)
	}

	return cw.BytesWritten(), totalSizeBytes, nil
}

type AsyncByteCountingWriter struct {
	W            io.Writer
	bytesWritten atomic.Int64
}

func (w *AsyncByteCountingWriter) Write(bs []byte) (int, error) {
	n, err := w.W.Write(bs)
	w.bytesWritten.Add(int64(n))
	return n, err
}

func (w *AsyncByteCountingWriter) BytesWritten() int64 {
	return w.bytesWritten.Load()
}
