package service

import (
	"context"

	v1 "coffee/api/coffee/v1"
	"coffee/app/coffee/internal/biz"
)

// CoffeeService implements the generated v1.CoffeeServer / CoffeeHTTPServer interfaces.
type CoffeeService struct {
	v1.UnimplementedCoffeeServer

	uc *biz.CoffeeUsecase
}

func NewCoffeeService(uc *biz.CoffeeUsecase) *CoffeeService {
	return &CoffeeService{uc: uc}
}

func (s *CoffeeService) Brew(ctx context.Context, in *v1.BrewRequest) (*v1.BrewReply, error) {
	id, err := s.uc.Brew(ctx, &biz.CoffeeOrder{
		Beans: in.GetBeans(),
		Size:  in.GetSize(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.BrewReply{InstanceId: id}, nil
}

func (s *CoffeeService) Check(ctx context.Context, in *v1.CheckRequest) (*v1.CheckReply, error) {
	st, err := s.uc.Check(ctx, in.GetInstanceId())
	if err != nil {
		return nil, err
	}
	return &v1.CheckReply{
		InstanceId: st.InstanceID,
		Status:     st.Status,
		Cup:        st.Cup,
	}, nil
}
