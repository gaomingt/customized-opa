package api

import (
	"context"
)

type Server struct {
	ctx context.Context

	Port int
}

func NewServer(ctx context.Context) (*Server, error) {
	server := &Server{
		ctx:  ctx,
		Port: 8080,
	}
	return server, nil
}
