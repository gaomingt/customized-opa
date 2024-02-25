package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gaomingt/customized-opa/api"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var (
	signals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT}
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), signals...)
	defer stop()

	eg, ctx := errgroup.WithContext(ctx)
	StartGrpcServer(ctx, eg)

	err := eg.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("server exited with error")
	}
}

func StartGrpcServer(ctx context.Context, eg *errgroup.Group) {
	s, err := api.NewServer(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start grpc server")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatal().Err(err).Int("port", s.Port).Msg("failed to listen on port")
	}

	grpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())
	reflection.Register(grpcServer)

	log.Info().Str("service", "grpc").Int("port", s.Port).Msg("started")

	eg.Go(func() error {
		err := grpcServer.Serve(lis)
		if err != nil {
			// handle manual stop
			if errors.Is(err, grpc.ErrServerStopped) {
				return nil
			}
			log.Error().Err(err).Msg("grpc server crashed")
			return err
		}
		return nil
	})

	// shut down gracefully
	eg.Go(func() error {
		<-ctx.Done()
		grpcServer.GracefulStop()
		return nil
	})
}

//func (s *Server) Check(ctx context.Context, in *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
//	return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
//}
//
//func (s *Server) Watch(in *health.HealthCheckRequest, _ health.Health_WatchServer) error {
//	// Example of how to register both methods but only implement the Check method.
//	return status.Error(codes.Unimplemented, "unimplemented")
//}
