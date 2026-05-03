package service

import (
	"context"

	v1 "greeter/api/greeter/orders/v1"
	"greeter/app/greeter/internal/biz"
)

// OrdersService implements the generated v1.OrdersServer / OrdersHTTPServer interfaces.
type OrdersService struct {
	v1.UnimplementedOrdersServer

	uc *biz.OrderUsecase
}

func NewOrdersService(uc *biz.OrderUsecase) *OrdersService {
	return &OrdersService{uc: uc}
}

func (s *OrdersService) StartOrder(ctx context.Context, in *v1.StartOrderRequest) (*v1.StartOrderReply, error) {
	id, err := s.uc.Start(ctx, &biz.Order{
		ItemName:  in.GetItemName(),
		TotalCost: in.GetTotalCost(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.StartOrderReply{InstanceId: id}, nil
}

func (s *OrdersService) GetOrder(ctx context.Context, in *v1.GetOrderRequest) (*v1.GetOrderReply, error) {
	st, err := s.uc.Get(ctx, in.GetInstanceId())
	if err != nil {
		return nil, err
	}
	return &v1.GetOrderReply{
		InstanceId:    st.InstanceID,
		RuntimeStatus: st.RuntimeStatus,
		Output:        st.Output,
	}, nil
}
