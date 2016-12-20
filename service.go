package addsvcdemo

import (
	"errors"
	"golang.org/x/net/context"
)

// Declare service middleware once here for logging and instrumenting
type Middleware func(Service) Service

type Service interface {
	Sum(ctx context.Context, x, y int) (int, error)
}

var (
	ErrIntOverflow = errors.New("integer overflow")
)

func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type statelessService struct{}

func NewStatelessService() Service {
	return statelessService{}
}

const (
	intMax = 1<<31 - 1
	intMin = -(intMax + 1)
)

// Sum implements Service
func (s statelessService) Sum(_ context.Context, x, y int) (int, error) {
	if (y > 0 && x > (intMax-y)) || (y < 0 && x < (intMin-y)) {
		return 0, ErrIntOverflow
	}
	return x + y, nil
}
