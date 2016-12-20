package addsvcdemo

import (
	"github.com/go-kit/kit/log"
	"golang.org/x/net/context"
	"time"
)

type serviceLoggingMiddleware struct {
	logger log.Logger
	next   Service
}

func ServiceLoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return serviceLoggingMiddleware{
			logger: logger,
			next:   next,
		}
	}
}

func (mw serviceLoggingMiddleware) Sum(ctx context.Context, x, y int) (v int, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "Sum",
			"x", x, "y", y, "result", v, "error", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.Sum(ctx, x, y)
}
