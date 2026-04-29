package biz

import "context"

type EventRepo interface {
	Publish(ctx context.Context, topic string, data []byte) error
}
