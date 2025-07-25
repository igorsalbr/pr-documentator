package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/igorsal/pr-documentator/internal/interfaces"
)

// MetricsMiddleware tracks HTTP request metrics
func MetricsMiddleware(metrics interfaces.MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap ResponseWriter to capture status code
			wrapped := &metricsResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			labels := map[string]string{
				"method":      r.Method,
				"endpoint":    r.URL.Path,
				"status_code": strconv.Itoa(wrapped.statusCode),
			}

			metrics.IncrementCounter("http_requests_total", labels)
			metrics.RecordDuration("http_request_duration_seconds", duration, labels)
		})
	}
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (mrw *metricsResponseWriter) WriteHeader(code int) {
	mrw.statusCode = code
	mrw.ResponseWriter.WriteHeader(code)
}

func (mrw *metricsResponseWriter) Write(b []byte) (int, error) {
	return mrw.ResponseWriter.Write(b)
}
