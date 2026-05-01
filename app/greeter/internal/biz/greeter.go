package biz

import (
	"context"

	v1 "greeter/api/greeter/helloworld/v1"

	"github.com/go-kratos/kratos/v2/errors"
)

var ErrUserNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")

type Greeter struct {
	ID    int64
	Hello string
}

type GreeterRepo interface {
	Save(context.Context, *Greeter) (*Greeter, error)
	FindByID(context.Context, int64) (*Greeter, error)
}

type GreeterUsecase struct {
	repo GreeterRepo
}

func NewGreeterUsecase(repo GreeterRepo) *GreeterUsecase {
	return &GreeterUsecase{repo: repo}
}

func (uc *GreeterUsecase) CreateGreeter(ctx context.Context, g *Greeter) (*Greeter, error) {
	return uc.repo.Save(ctx, g)
}

func (uc *GreeterUsecase) GetGreeter(ctx context.Context, id int64) (*Greeter, error) {
	return uc.repo.FindByID(ctx, id)
}
