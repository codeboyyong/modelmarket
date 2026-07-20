package httpapi

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
)

type requestMetric struct {
	Count        int64
	Errors       int64
	LatencyMS    int64
	MaxLatencyMS int64
}

type metricWriter struct {
	http.ResponseWriter
	status int
}

func (w *metricWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *metricWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func (w *metricWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *metricWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijacking unsupported")
	}
	return hijacker.Hijack()
}

func (a *App) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &metricWriter{ResponseWriter: w}
		next.ServeHTTP(wrapped, r)
		status := wrapped.status
		if status == 0 {
			status = http.StatusOK
		}
		latency := time.Since(start).Milliseconds()
		key := r.Method + " " + routeMetricPath(r.URL.Path)
		a.metricsMu.Lock()
		if a.metrics == nil {
			a.metrics = map[string]*requestMetric{}
		}
		metric := a.metrics[key]
		if metric == nil {
			metric = &requestMetric{}
			a.metrics[key] = metric
		}
		metric.Count++
		metric.LatencyMS += latency
		if latency > metric.MaxLatencyMS {
			metric.MaxLatencyMS = latency
		}
		if status >= 400 {
			metric.Errors++
		}
		a.metricsMu.Unlock()
		if a.Logger != nil {
			args := []any{"request_id", w.Header().Get("X-Request-ID"), "method", r.Method, "path", r.URL.Path, "status", status, "latency_ms", latency}
			if status >= 500 {
				a.Logger.Error("http_request_failed", args...)
			} else if status >= 400 {
				a.Logger.Warn("http_request_rejected", args...)
			} else {
				a.Logger.Info("http_request", args...)
			}
		}
	})
}

func routeMetricPath(path string) string {
	for _, prefix := range []string{"/api/v1/conversations/", "/api/v1/assets/", "/api/v1/api-keys/", "/api/v1/mock-s3/"} {
		if strings.HasPrefix(path, prefix) {
			return prefix + ":id"
		}
	}
	return path
}

func (a *App) serveMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	a.metricsMu.Lock()
	keys := make([]string, 0, len(a.metrics))
	for key := range a.metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := []string{"# HELP model_market_http_requests_total HTTP requests.", "# TYPE model_market_http_requests_total counter"}
	for _, key := range keys {
		metric := *a.metrics[key]
		method, path, _ := strings.Cut(key, " ")
		labels := fmt.Sprintf(`method=%q,path=%q`, method, path)
		lines = append(lines,
			fmt.Sprintf("model_market_http_requests_total{%s} %d", labels, metric.Count),
			fmt.Sprintf("model_market_http_errors_total{%s} %d", labels, metric.Errors),
			fmt.Sprintf("model_market_http_latency_milliseconds_sum{%s} %d", labels, metric.LatencyMS),
			fmt.Sprintf("model_market_http_latency_milliseconds_max{%s} %d", labels, metric.MaxLatencyMS),
		)
	}
	a.metricsMu.Unlock()
	_, _ = fmt.Fprintln(w, strings.Join(lines, "\n"))
}
