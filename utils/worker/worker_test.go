package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/prometheus"
)

func TestWorker(t *testing.T) {
	for k, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cancel()
				w.WriteHeader(200)
				w.Write([]byte("Hello World"))
				if r.Method != method {
					t.Logf("Expected %+v; Got %+v", method, r.Method)
					t.FailNow()
				}
			}))
			defer ts.Close()
			defer cancel()
			worker := NewWorker(ctx, &interfaces.WorkerOpts{
				URL:     ts.URL,
				Method:  method,
				Headers: map[string]string{},
				Follow:  false,
			})
			worker.Do()
			metrics := prometheus.Metrics.Get()
			k++
			if metrics.RequestsTotal != float64(k) {
				t.Logf("Expected %+v; Got %+v", k, metrics.RequestsTotal)
				t.FailNow()
			}
		})
	}
}
