package logmiddleware

import (
	"context"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/inconshreveable/log15"
)

type ctxKey string

var (
	lgrContextKey = ctxKey("lgr")
)

func LgrFromContext(ctx context.Context) log15.Logger {
	lgrI := ctx.Value(lgrContextKey)
	if lgrI == nil {
		return log15.New()
	}
	return lgrI.(log15.Logger).New()
}

func WithLgrContext(ctx context.Context, lgr log15.Logger) context.Context {
	return context.WithValue(ctx, lgrContextKey, lgr)
}

func New(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := *r.URL
		host := r.Host

		lgr := log15.New("url", url.String(), "host", host, "remote_addr", r.RemoteAddr)

		if reqId := r.Header.Get("X-LambdaHttp-Aws-Request-Id"); reqId != "" {
			lgr = lgr.New("aws_request_id", reqId)
		}

		childCtx := WithLgrContext(r.Context(), lgr)
		childReq := r.WithContext(childCtx)

		metrics := httpsnoop.CaptureMetrics(next, w, childReq)

		lgr.Info("request", "status", metrics.Code, "duration_ms", metrics.Duration.Milliseconds(), "resp_size", metrics.Written, "method", r.Method, "proto", r.Proto)
	})
}
