package log

import (
	"time"

	"github.com/rs/zerolog"
)

const (
	grpcProtocol = "grpc"
	httpProtocol = "http"
)

func loggerHelper(e *zerolog.Event, protocol, httpMethod, path, statusText string, statusCode int, duration time.Duration) {
	if path == "/health" || path == "/grpc.health.v1.Health/Check" {
		return
	}
	if protocol == httpProtocol {
		e.Str("method", httpMethod)
	}

	e.
		Str("protocol", protocol).
		Str("path", path).
		Str("status_text", statusText).
		Int("status_code", statusCode).
		Dur("duration", duration).
		Msgf("processed a %s request", protocol)
}
