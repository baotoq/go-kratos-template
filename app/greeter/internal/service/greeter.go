package service

import (
	"context"

	v1 "greeter/api/greeter/helloworld/v1"
	"greeter/app/greeter/internal/biz"
)

type GreeterService struct {
	v1.UnimplementedGreeterServer

	uc *biz.GreeterUsecase
}

func NewGreeterService(uc *biz.GreeterUsecase) *GreeterService {
	return &GreeterService{uc: uc}
}

func (s *GreeterService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	g, err := s.uc.CreateGreeter(ctx, &biz.Greeter{Hello: in.Name})
	if err != nil {
		return nil, err
	}
	return &v1.HelloReply{Message: "Hello " + g.Hello}, nil
}

func (s *GreeterService) CreateGreeter(ctx context.Context, in *v1.CreateGreeterRequest) (*v1.CreateGreeterReply, error) {
	g, err := s.uc.CreateGreeter(ctx, &biz.Greeter{Hello: in.Hello})
	if err != nil {
		return nil, err
	}
	return &v1.CreateGreeterReply{Id: int64(g.ID), Hello: g.Hello}, nil
}

func (s *GreeterService) GetGreeter(ctx context.Context, in *v1.GetGreeterRequest) (*v1.GetGreeterReply, error) {
	g, err := s.uc.GetGreeter(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetGreeterReply{Id: int64(g.ID), Hello: g.Hello}, nil
}
