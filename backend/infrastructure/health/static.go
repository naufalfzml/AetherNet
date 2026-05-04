package health

import (
	"context"
	"errors"
)

type StaticClient struct {
	Healthy bool
	Message string
}

func (c StaticClient) Health(context.Context) error {
	if c.Healthy {
		return nil
	}
	if c.Message == "" {
		return errors.New("unhealthy")
	}
	return errors.New(c.Message)
}
