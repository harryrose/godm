package db

import (
	"cmp"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/harryrose/godm/queue-service/queue"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"regexp"
	"strings"
	"time"
)

const (
	QueueBucket     = "queues"
	QueueMetaKey    = "meta"
	ItemsBucket     = "items"
	FinishedBucket  = "finished"
	idSeparator     = ":"
	claimTTL        = time.Second * 30
	MaxPageSize     = 100
	DefaultPageSize = 50
)

func NewBolt(path string) (*Bolt, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	out := &Bolt{
		db: db,
	}

	return out, nil
}

type Bolt struct {
	now func() time.Time
	db  *bolt.DB
}

func ensureBucket(parent interface {
	Bucket([]byte) *bolt.Bucket
	CreateBucket([]byte) (*bolt.Bucket, error)
}, newName string) (*bolt.Bucket, error) {
	buc, err := parent.CreateBucket([]byte(newName))
	if err != nil {
		if !errors.Is(err, bolt.ErrBucketExists) {
			return nil, err
		}
		buc = parent.Bucket([]byte(newName))
	}
	return buc, nil
}

func (b *Bolt) CreateQueue(name string) (*Queue, error) {
	var out Queue
	err := b.db.Update(func(tx *bolt.Tx) error {
		queues, err := ensureBucket(tx, QueueBucket)
		if err != nil {
			return fmt.Errorf("error creating bucket %v: %w", QueueBucket, err)
		}
		key := sanitiseQueueName(name)

		queuebucket, err := queues.CreateBucket([]byte(key))
		if err != nil {
			if errors.Is(err, bolt.ErrBucketExists) {
				return ErrConflict{}
			}
			return fmt.Errorf("error creating queue bucket, %v: %w", key, err)
		}

		items, err := ensureBucket(queuebucket, ItemsBucket)
		if err != nil {
			return fmt.Errorf("error creating items bucket: %w", err)
		}

		_, err = ensureBucket(queuebucket, FinishedBucket)
		if err != nil {
			return fmt.Errorf("error creating abandoned bucket: %w", err)
		}

		if err := items.SetSequence(10); err != nil {
			return fmt.Errorf("error setting sequence: %w", err)
		}

		out.Id = key
		out.Name = name
		out.Timestamp = &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
		}

		marsh, err := proto.Marshal(&out)
		if err != nil {
			return fmt.Errorf("error marshalling queue object: %w", err)
		}
		if err := queuebucket.Put([]byte(QueueMetaKey), marsh); err != nil {
			return fmt.Errorf("error putting queue object: %w", err)
		}

		return nil
	})
	return &out, err
}

func (b *Bolt) EnqueueItem(queue, source, destination, category string) (string, error) {
	var out string
	err := b.db.Update(func(tx *bolt.Tx) error {
		items, err := b.getQueueItemsBucket(tx, queue)
		if err != nil {
			return err
		}
		id, err := items.NextSequence()
		if err != nil {
			return fmt.Errorf("error getting next sequence: %w", err)
		}
		q := sanitiseQueueName(queue)
		itemID := fmt.Sprintf("%s"+idSeparator+"%020d", q, id)
		out = itemID
		var item Item
		item.Id = itemID
		item.Source = &Target{
			Url: source,
		}
		item.Destination = &Target{
			Url: destination,
		}
		item.Category = &Category{
			Id: category,
		}

		ibs, err := proto.Marshal(&item)
		if err != nil {
			return fmt.Errorf("error marshalling item: %w", err)
		}
		if err := items.Put([]byte(itemID), ibs); err != nil {
			return fmt.Errorf("error adding item: %w", err)
		}
		return nil
	})
	return out, err
}

func (b *Bolt) SetItemState(id string, state queue.ItemState_State, bytesDownloaded uint64, totalSizeBytes uint64, err error) error {
	switch state {
	case queue.ItemState_ITEM_STATE_UNSPECIFIED:
		return fmt.Errorf("state was not specified")

	case queue.ItemState_ITEM_STATE_FAILED:
		return b.FailItem(id, bytesDownloaded, totalSizeBytes, err)

	case queue.ItemState_ITEM_STATE_COMPLETE:
		return b.CompleteItem(id, totalSizeBytes)

	case queue.ItemState_ITEM_STATE_DOWNLOADING:
		return b.SetProgress(id, bytesDownloaded, totalSizeBytes)

	default:
		return fmt.Errorf("unrecognised state: %v, %v", int32(state), queue.ItemState_State_name[int32(state)])
	}

}

func (b *Bolt) SetProgress(id string, bytesDownloaded uint64, totalSizeBytes uint64) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		q, err := queueKeyFromItemID(id)
		if err != nil {
			return ErrInvalid{}
		}

		items, err := b.getQueueItemsBucket(tx, q)
		if err != nil {
			return err
		}

		bs := items.Get([]byte(id))
		if len(bs) == 0 {
			return ErrNotFound{}
		}

		var item Item
		if err := proto.Unmarshal(bs, &item); err != nil {
			return fmt.Errorf("error unmarshalling item: %w", err)
		}
		item.TotalSizeBytes = totalSizeBytes
		item.DownloadedBytes = bytesDownloaded
		item.ClaimExpiry = timestamppb.New(time.Now().Add(claimTTL))

		bs, err = proto.Marshal(&item)
		if err != nil {
			return fmt.Errorf("error marshalling item: %w", err)
		}
		if err := items.Put([]byte(id), bs); err != nil {
			return fmt.Errorf("error storing update: %w", err)
		}
		return nil
	})
}

func (b *Bolt) CompleteItem(id string, totalSizeBytes uint64) error {
	return b.moveItemToFinished(id, totalSizeBytes, totalSizeBytes, FinishedItem_ITEM_STATE_SUCCESS, "")
}

func (b *Bolt) FailItem(id string, downloadedBytes, totalSizeBytes uint64, err error) error {
	return b.moveItemToFinished(id, downloadedBytes, totalSizeBytes, FinishedItem_ITEM_STATE_FAILED, err.Error())
}

func (b *Bolt) CancelItem(id string) error {
	return b.moveItemToFinished(id, 0, 0, FinishedItem_ITEM_STATE_CANCELLED, "cancelled by user")
}

func (b *Bolt) GetQueueItems(queueID string, startKey string, pageSize uint) (queueItems []Item, nextKey string, err error) {
	return getQueueItems(b, func() *Item {
		return &Item{}
	}, func(t *Item) Item {
		return *t
	}, queueID, ItemsBucket, startKey, pageSize)
}

func (b *Bolt) GetFinishedItems(queueID string, startKey string, pageSize uint) (queueItems []FinishedItem, nextKey string, err error) {
	return getQueueItems(b, func() *FinishedItem {
		return &FinishedItem{}
	}, func(t *FinishedItem) FinishedItem {
		return *t
	}, queueID, FinishedBucket, startKey, pageSize)
}

func (b *Bolt) ClearHistory(queueID string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		fin, err := b.getFinishedItemsBucket(tx, queueID)
		if err != nil {
			return err
		}
		return fin.ForEach(func(k, _ []byte) error {
			if err := fin.Delete(k); err != nil {
				log.Printf("error deleting %v: %v\n", string(k), err)
			}
			return nil
		})
	})
}

func getQueueItems[T proto.Message, V any](b *Bolt, newT func() T, tToV func(T) V, queueID string, itemsName string, startKey string, pageSize uint) (queueItems []V, nextKey string, err error) {
	pageSize = min(defaultIfEmpty(DefaultPageSize, pageSize), MaxPageSize)
	queueItems = make([]V, 0, pageSize)
	nextKey = ""
	err = b.db.View(func(tx *bolt.Tx) error {
		items, err := b.getQueueInnerBucket(tx, queueID, itemsName)
		if err != nil {
			return fmt.Errorf("queue %s: %w", queueID, err)
		}

		limitReached := errors.New("limit reached")
		itemCount := uint(0)
		err = items.ForEach(func(k, v []byte) error {
			sk := string(k)
			if startKey != "" && sk < startKey {
				return nil
			}
			if itemCount >= pageSize {
				nextKey = sk
				return limitReached
			}
			item := newT()
			if err := proto.Unmarshal(v, item); err != nil {
				return fmt.Errorf("unmarshalling item: %w", err)
			}
			queueItems = append(queueItems, tToV(item))
			return nil
		})

		switch {
		case errors.Is(err, limitReached):
			return nil
		case err == nil:
			return nil
		default:
			return err
		}
	})
	return
}

func (b *Bolt) ClaimNextItem(queue string) (*Item, error) {
	var nextItem Item
	var nextItemKey []byte
	itemFound := errors.New("found")

	err := b.db.Update(func(tx *bolt.Tx) error {
		items, err := b.getQueueItemsBucket(tx, queue)
		if err != nil {
			return err
		}
		err = items.ForEach(func(k, v []byte) error {
			var item Item
			if err := proto.Unmarshal(v, &item); err != nil {
				return fmt.Errorf("error unmarshalling item: %w", err)
			}

			if item.ClaimExpiry.AsTime().Before(time.Now()) {
				// the item's claim is expired, so return it
				nextItem = item
				nextItemKey = k
				return itemFound
			}
			return nil
		})

		if !errors.Is(err, itemFound) {
			// we either didn't find anything or something bad happened
			return err
		}

		newExpiryTime := time.Now().Add(claimTTL)
		nextItem.ClaimExpiry = timestamppb.New(newExpiryTime)
		enc, err := proto.Marshal(&nextItem)
		if err != nil {
			return fmt.Errorf("error marshalling claimed item: %w", err)
		}
		err = items.Put(nextItemKey, enc)
		if err != nil {
			return fmt.Errorf("error storing claimed item: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(nextItemKey) == 0 {
		// we didn't find anything
		return nil, nil
	}
	return &nextItem, nil
}

func (b *Bolt) moveItemToFinished(id string, downloadedBytes uint64, totalSizeBytes uint64, state FinishedItem_State, message string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		q, err := queueKeyFromItemID(id)
		if err != nil {
			return ErrInvalid{}
		}

		act, err := b.getQueueItemsBucket(tx, q)
		if err != nil {
			return err
		}
		fin, err := b.getFinishedItemsBucket(tx, q)
		if err != nil {
			return err
		}

		var item Item
		itembs := act.Get([]byte(id))
		if len(itembs) == 0 {
			return ErrNotFound{}
		}
		if err := proto.Unmarshal(itembs, &item); err != nil {
			return fmt.Errorf("unable to unmarshal item: %w", err)
		}

		key, err := orderedKey()
		if err != nil {
			return fmt.Errorf("unable to generate key: %w", err)
		}

		finished := &FinishedItem{
			State:           state,
			TotalSizeBytes:  totalSizeBytes,
			DownloadedBytes: downloadedBytes,
			Timestamp:       timestamppb.New(time.Now()),
			Message:         message,
			Item:            &item,
		}
		fbs, err := proto.Marshal(finished)
		if err != nil {
			return fmt.Errorf("unable to marshal finished item: %w", err)
		}
		if err := fin.Put([]byte(key), fbs); err != nil {
			return fmt.Errorf("unable to write finished item: %w", err)
		}
		if err := act.Delete([]byte(id)); err != nil {
			return fmt.Errorf("inable to delete item from queue: %w", err)
		}
		return nil
	})
}

func orderedKey() (string, error) {
	const size = 128
	out := make([]byte, size)
	n, err := rand.Read(out)
	if err != nil {
		return "", fmt.Errorf("error reading random data: %w", err)
	}
	if n != size {
		return "", fmt.Errorf("did not fill buffer: %w", err)
	}

	now := uint64(time.Now().UnixNano())
	binary.BigEndian.PutUint64(out, now)

	return hex.EncodeToString(out), nil
}

func queueKeyFromItemID(id string) (string, error) {
	col := strings.Index(id, idSeparator)
	if col <= 1 { // 1 because we can't have 0-length queue names
		return "", ErrInvalid{}
	}
	return id[:col], nil
}

func (b *Bolt) getFinishedItemsBucket(tx *bolt.Tx, queue string) (*bolt.Bucket, error) {
	return b.getQueueInnerBucket(tx, queue, FinishedBucket)
}

func (b *Bolt) getQueueItemsBucket(tx *bolt.Tx, queue string) (*bolt.Bucket, error) {
	return b.getQueueInnerBucket(tx, queue, ItemsBucket)
}

func (b *Bolt) getQueueInnerBucket(tx *bolt.Tx, queue string, itemsname string) (*bolt.Bucket, error) {
	queuesBucket := tx.Bucket([]byte(QueueBucket))
	if queuesBucket == nil {
		// it can't exist because apparently we've not even created the collection yet!
		return nil, ErrNotFound{}
	}
	queueKey := sanitiseQueueName(queue)
	queueBucket := queuesBucket.Bucket([]byte(queueKey))
	if queueBucket == nil {
		return nil, ErrNotFound{}
	}
	items := queueBucket.Bucket([]byte(itemsname))
	if items == nil {
		return nil, ErrNotFound{}
	}
	return items, nil
}

var invalidQueueChars = regexp.MustCompile("[^a-zA-Z0-9_-]")

func sanitiseQueueName(in string) string {
	return invalidQueueChars.ReplaceAllLiteralString(in, "_")
}

func defaultIfEmpty[T comparable](def T, act T) T {
	var empty T
	if act == empty {
		return def
	}
	return act
}

func min[T cmp.Ordered](a, b T) T {
	if a > b {
		return b
	}
	return a
}
