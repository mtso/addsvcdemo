package addsvcdemo

import (
	"github.com/go-kit/kit/metrics"
	"golang.org/x/net/context"
)

type serviceInstrumentingMiddleware struct {
	ints metrics.Counter
	next Service
}

func ServiceInstrumentingMiddleware(ints metrics.Counter) Middleware {
	return func(next Service) Service {
		return serviceInstrumentingMiddleware{
			ints: ints,
			next: next,
		}
	}
}

func (mw serviceInstrumentingMiddleware) Sum(ctx context.Context, x, y int) (int, error) {
	v, err := mw.next.Sum(ctx, x, y)
	mw.ints.Add(float64(v))
	return v, err
}
