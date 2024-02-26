package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gaomingt/customized-opa/api"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	signals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT}
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), signals...)
	defer stop()

	eg, ctx := errgroup.WithContext(ctx)
	startGrpcServer(ctx, eg)
	startGatewayServer(ctx, eg, 8080)

	err := eg.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("server exited with error")
	}
}

func startGrpcServer(ctx context.Context, eg *errgroup.Group) {
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

	log.Info().Str("service", "grpc server").Int("port", s.Port).Msg("started")

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
		log.Info().Str("service", "grpc server").Msg("gracefully shutdown")
		return nil
	})
}

func startGatewayServer(ctx context.Context, eg *errgroup.Group, port int) {
	grpcGateway := runtime.NewServeMux(
		// configure http response body
		runtime.WithMarshalerOption(
			runtime.MIMEWildcard, &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					UseProtoNames: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		),
		// configure authorization header
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch key {
			case "authorization":
				return key, true
			default:
				return runtime.DefaultHeaderMatcher(key)
			}
		}),
	)

	mux := http.NewServeMux()
	mux.Handle("/", grpcGateway)

	httpServer := &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", port),
	}

	eg.Go(func() error {
		log.Info().Str("service", "grpc gateway").Int("port", port).Msg("started")
		err := httpServer.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			log.Error().Err(err).Msg("HTTP gateway server failed to serve")
			return err
		}
		return nil
	})

	eg.Go(func() error {
		<-ctx.Done()
		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("failed to shutdown HTTP gateway server")
			return err
		}

		log.Info().Str("service", "grpc gateway").Msg("gracefully shutdown")
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
