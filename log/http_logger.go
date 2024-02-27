package log

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// HttpResponseRecord is a struct to store metadata for logging
type HttpResponseRecord struct {
	http.ResponseWriter
	StatusCode int
	Body       []byte
}

// WriteHeader is a wrapper around standard ResponseWriter.WriteHeader
func (rec *HttpResponseRecord) WriteHeader(statusCode int) {
	rec.StatusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

// Write is a wrapper around standard ResponseWriter.Write
func (rec *HttpResponseRecord) Write(body []byte) (int, error) {
	rec.Body = body
	return rec.ResponseWriter.Write(body)
}

func HttpLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// trick to reuse the request body
		reqBody, err := io.ReadAll(r.Body)
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		ctx := r.Context()
		rec := &HttpResponseRecord{
			ResponseWriter: w,
			StatusCode:     http.StatusOK,
		}
		handler.ServeHTTP(rec, r.WithContext(ctx))
		duration := time.Since(start)

		event := log.Info()
		if rec.StatusCode != http.StatusOK {
			event = log.Error().Err(err).RawJSON("body", rec.Body)
		}

		loggerHelper(event, httpProtocol, r.Method, r.RequestURI, http.StatusText(rec.StatusCode), rec.StatusCode, duration)
	})
}
