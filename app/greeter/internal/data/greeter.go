package data

import (
	"context"

	"greeter/app/greeter/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type greeterRepo struct {
	data *Data
	log  *log.Helper
}

func NewGreeterRepo(data *Data, logger log.Logger) biz.GreeterRepo {
	return &greeterRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *greeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	out, err := r.data.db.Greeter.Create().SetHello(g.Hello).Save(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.Greeter{ID: int64(out.ID), Hello: out.Hello}, nil
}

func (r *greeterRepo) FindByID(ctx context.Context, id int64) (*biz.Greeter, error) {
	out, err := r.data.db.Greeter.Get(ctx, int(id))
	if err != nil {
		return nil, err
	}
	return &biz.Greeter{ID: int64(out.ID), Hello: out.Hello}, nil
}
