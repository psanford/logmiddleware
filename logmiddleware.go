package logmiddleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/felixge/httpsnoop"
)

type ctxKey string

var (
	lgrContextKey = ctxKey("lgr")
)

func LgrFromContext(ctx context.Context) *slog.Logger {
	lgrI := ctx.Value(lgrContextKey)
	if lgrI == nil {
		return slog.With()
	}
	return lgrI.(*slog.Logger)
}

func WithLgrContext(ctx context.Context, lgr *slog.Logger) context.Context {
	return context.WithValue(ctx, lgrContextKey, lgr)
}

func New(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := *r.URL
		host := r.Host

		lgr := slog.With("url", url.String(), "host", host, "remote_addr", r.RemoteAddr)

		if reqId := r.Header.Get("X-LambdaHttp-Aws-Request-Id"); reqId != "" {
			lgr = lgr.With("aws_request_id", reqId)
		}

		childCtx := WithLgrContext(r.Context(), lgr)
		childReq := r.WithContext(childCtx)

		metrics := httpsnoop.CaptureMetrics(next, w, childReq)

		lgr.Info("request", "status", metrics.Code, "duration_ms", metrics.Duration.Milliseconds(), "resp_size", metrics.Written, "method", r.Method, "proto", r.Proto, "user_agent", r.UserAgent(), "referer", r.Referer())
	})
}
