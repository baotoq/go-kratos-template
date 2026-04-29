package service

import (
	"context"

	"github.com/dapr/go-sdk/service/common"
	"github.com/go-kratos/kratos/v2/log"
)

type EventService struct {
	log *log.Helper
}

func NewEventService(logger log.Logger) *EventService {
	return &EventService{log: log.NewHelper(logger)}
}

func (s *EventService) HandleEvent(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	s.log.WithContext(ctx).Infof("event: id=%s topic=%s data=%s", e.ID, e.Topic, e.RawData)
	return false, nil
}
