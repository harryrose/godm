package queue_service

import (
	"context"
	"errors"
	db2 "github.com/harryrose/godm/queue-service/db"
	"github.com/harryrose/godm/queue-service/queue"
	"github.com/harryrose/godm/queue-service/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type Service struct {
	DB *db2.Bolt
	rpc.UnimplementedQueueServiceServer
}

func coerceDBError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.As(err, &db2.ErrNotFound{}):
		return status.Error(codes.NotFound, err.Error())

	case errors.As(err, &db2.ErrInvalid{}):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.As(err, &db2.ErrConflict{}):
		return status.Error(codes.AlreadyExists, err.Error())

	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func (s *Service) CreateQueue(ctx context.Context, in *rpc.CreateQueueInput) (*rpc.CreateQueueResult, error) {
	if in == nil || len(in.Name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "name must be provided and non-zero length")
	}

	_, err := s.DB.CreateQueue(in.Name)
	if err != nil {
		return nil, coerceDBError(err)
	}

	return &rpc.CreateQueueResult{}, nil
}

func (s *Service) EnqueueItem(ctx context.Context, in *rpc.EnqueueItemInput) (*rpc.EnqueueItemResult, error) {
	if in == nil {
		return nil, status.Errorf(codes.InvalidArgument, "no parameters provided")
	}
	if in.Item == nil {
		return nil, status.Errorf(codes.InvalidArgument, "item must be provided")
	}
	if in.Item.Source == nil {
		return nil, status.Errorf(codes.InvalidArgument, "item source must be provided")
	}
	if len(in.Item.Source.Url) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "item source url must be provided and non-empty")
	}
	if in.Item.Destination == nil {
		return nil, status.Errorf(codes.InvalidArgument, "item destination must be provided")
	}
	if len(in.Item.Destination.Url) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "item destination url must be provided and non-empty")
	}
	if in.Queue == nil {
		return nil, status.Errorf(codes.InvalidArgument, "queue must be provided")
	}
	if len(in.Queue.Id) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "queue id must be provided and non-empty")
	}

	res, err := s.DB.EnqueueItem(in.Queue.Id, in.Item.Source.Url, in.Item.Destination.Url, in.Item.Category.Id.Id)
	if err != nil {
		return nil, coerceDBError(err)
	}
	return &rpc.EnqueueItemResult{Id: &queue.Identifier{Id: res}}, nil
}

func (s *Service) CancelItem(ctx context.Context, in *rpc.CancelItemInput) (*rpc.CancelItemResult, error) {
	if in == nil {
		return nil, status.Errorf(codes.InvalidArgument, "no parameters provided")
	}
	if in.Item == nil {
		return nil, status.Errorf(codes.InvalidArgument, "item must be provided")
	}
	if len(in.Item.Id) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "item id must be provided and non-empty")
	}
	err := s.DB.CancelItem(in.Item.Id)
	return &rpc.CancelItemResult{}, coerceDBError(err)
}

func (s *Service) GetFinishedItems(ctx context.Context, in *rpc.GetFinishedItemsInput) (*rpc.GetFinishedItemsResult, error) {
	if in == nil {
		return nil, status.Errorf(codes.InvalidArgument, "no parameters provided")
	}
	if in.Queue == nil {
		return nil, status.Errorf(codes.InvalidArgument, "queue must be provided")
	}
	if len(in.Queue.Id) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "queue id must be provided")
	}

	limit := uint(0)
	startKey := ""

	items, next, err := s.DB.GetFinishedItems(in.Queue.Id, startKey, limit)
	if err != nil {
		return nil, coerceDBError(err)
	}
	out := rpc.GetFinishedItemsResult{
		Items: make([]*rpc.IdentifiedQueueItemWithState, len(items)),
		Pagination: &rpc.PaginationParameters{
			Limit: uint32(limit),
			Next: &queue.Identifier{
				Id: next,
			},
		},
	}

	for idx, _ := range items {
		var itemState queue.ItemState_State
		switch items[idx].State {
		case db2.FinishedItem_ITEM_STATE_SUCCESS:
			itemState = queue.ItemState_ITEM_STATE_COMPLETE
		case db2.FinishedItem_ITEM_STATE_FAILED:
			itemState = queue.ItemState_ITEM_STATE_FAILED
		case db2.FinishedItem_ITEM_STATE_CANCELLED:
			itemState = queue.ItemState_ITEM_STATE_FAILED
		default:
			itemState = queue.ItemState_ITEM_STATE_UNSPECIFIED
		}

		out.Items[idx] = &rpc.IdentifiedQueueItemWithState{
			Id: &queue.Identifier{
				Id: items[idx].Item.Id,
			},
			Item: &queue.Item{
				Source: &queue.Target{
					Url: items[idx].Item.Source.Url,
				},
				Destination: &queue.Target{
					Url: items[idx].Item.Destination.Url,
				},
				Category: &queue.Category{Id: &queue.Identifier{Id: items[idx].Item.Category.Id}},
			},
			State: &queue.ItemState{
				State:           itemState,
				TotalSizeBytes:  items[idx].TotalSizeBytes,
				DownloadedBytes: items[idx].DownloadedBytes,
				Message:         "",
			},
			Updated: items[idx].Timestamp,
		}
	}
	return &out, nil
}

func (s *Service) GetQueueItems(ctx context.Context, in *rpc.GetQueueItemsInput) (*rpc.GetQueueItemsResult, error) {
	if in == nil {
		return nil, status.Errorf(codes.InvalidArgument, "no parameters provided")
	}
	if in.Queue == nil {
		return nil, status.Errorf(codes.InvalidArgument, "queue must be provided")
	}
	if len(in.Queue.Id) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "queue id must be provided")
	}

	limit := uint(0)
	startKey := ""

	items, next, err := s.DB.GetQueueItems(in.Queue.Id, startKey, limit)
	if err != nil {
		return nil, coerceDBError(err)
	}
	out := rpc.GetQueueItemsResult{
		Items: make([]*rpc.IdentifiedQueueItemWithState, len(items)),
		Pagination: &rpc.PaginationParameters{
			Limit: uint32(limit),
			Next: &queue.Identifier{
				Id: next,
			},
		},
	}
	for idx, _ := range items {
		state := queue.ItemState_ITEM_STATE_QUEUED
		if items[idx].ClaimExpiry.AsTime().After(time.Now()) {
			state = queue.ItemState_ITEM_STATE_DOWNLOADING
		}

		out.Items[idx] = &rpc.IdentifiedQueueItemWithState{
			Id: &queue.Identifier{
				Id: items[idx].Id,
			},
			Item: &queue.Item{
				Source: &queue.Target{
					Url: items[idx].Source.Url,
				},
				Destination: &queue.Target{
					Url: items[idx].Destination.Url,
				},
				Category: &queue.Category{Id: &queue.Identifier{Id: items[idx].Category.Id}},
			},
			State: &queue.ItemState{
				State:           state,
				TotalSizeBytes:  items[idx].TotalSizeBytes,
				DownloadedBytes: items[idx].DownloadedBytes,
				Message:         "",
			},
			Updated: timestamppb.New(time.Now()),
		}
	}
	return &out, nil
}

func (s *Service) ClaimNextItem(ctx context.Context, in *rpc.ClaimNextItemInput) (*rpc.ClaimNextItemResult, error) {
	if in == nil {
		return nil, status.Errorf(codes.InvalidArgument, "no parameters provided")
	}
	if in.Queue == nil {
		return nil, status.Errorf(codes.InvalidArgument, "queue must be provided")
	}
	if len(in.Queue.Id) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "queue id must be provided and non-empty")
	}
	item, err := s.DB.ClaimNextItem(in.Queue.Id)
	if err != nil {
		return nil, coerceDBError(err)
	}
	if item == nil {
		// no more items
		return &rpc.ClaimNextItemResult{}, nil
	}

	return &rpc.ClaimNextItemResult{
		Id: &queue.Identifier{
			Id: item.Id,
		},
		Item: &queue.Item{
			Source:      &queue.Target{Url: item.Source.Url},
			Destination: &queue.Target{Url: item.Destination.Url},
			Category: &queue.Category{
				Id: &queue.Identifier{
					Id: item.Category.Id,
				},
			},
		},
	}, coerceDBError(err)
}

func (s *Service) SetItemState(ctx context.Context, in *rpc.SetItemStateInput) (*rpc.SetItemStateResult, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "no parameters provided")
	}
	if in.Item == nil {
		return nil, status.Error(codes.InvalidArgument, "item must be provided")
	}
	if len(in.Item.Id) == 0 {
		return nil, status.Error(codes.InvalidArgument, "item id must be provided and non-empty")
	}
	if in.State == nil {
		return nil, status.Error(codes.InvalidArgument, "state must be provided")
	}
	if in.State.State == queue.ItemState_ITEM_STATE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "state state must be provided and not unspecified")
	}
	err := s.DB.SetItemState(in.Item.Id, in.State.State, in.State.DownloadedBytes, in.State.TotalSizeBytes, errors.New(in.State.Message))
	return &rpc.SetItemStateResult{}, coerceDBError(err)
}

func (s *Service) ClearHistory(ctx context.Context, in *rpc.ClearHistoryInput) (*rpc.ClearHistoryResult, error) {
	if in == nil {
		return nil, status.Errorf(codes.InvalidArgument, "no parameters provided")
	}
	if in.Queue == nil {
		return nil, status.Errorf(codes.InvalidArgument, "queue must be provided")
	}
	if len(in.Queue.Id) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "queue id must be provided and non-empty")
	}

	err := s.DB.ClearHistory(in.Queue.Id)
	return &rpc.ClearHistoryResult{}, coerceDBError(err)
}
