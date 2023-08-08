package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/prometheus"
)

func TestHTTPWorker(t *testing.T) {
	for k, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(999)
				w.Write([]byte("Hello World"))
				if r.Method != method {
					t.Logf("Expected %+v; Got %+v", method, r.Method)
					t.FailNow()
				}
				body, _ := io.ReadAll(r.Body)
				if string(body) != "Hello World" {
					t.Logf("Expected %+v; Got %+v", "Hello World", string(body))
					t.FailNow()
				}
			}))
			defer ts.Close()
			worker := NewHTTPWorker(ctx, &interfaces.HTTPOpts{
				URL:     ts.URL,
				Method:  method,
				Headers: map[string]string{},
				Follow:  false,
			}, []byte("Hello World"))
			err := worker.Do()
			if err != nil && !errors.Is(err, ErrHTTPStatus) {
				t.Logf("Expected %+v; Got %+v", nil, err)
				t.FailNow()
			}
			metrics := prometheus.Metrics.Get()
			k++
			if metrics.ResponseCodes["999"] != float64(k) {
				t.Logf("Expected %+v; Got %+v", k, metrics.ResponseCodes["999"])
				t.FailNow()
			}
		})
	}
}

func TestNetWorker(t *testing.T) {
	for k, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte("Hello World"))
				if r.Method != method {
					t.Logf("Expected %+v; Got %+v", method, r.Method)
					t.FailNow()
				}
			}))
			u, _ := url.Parse(ts.URL)
			defer ts.Close()
			worker := NewNetWorker(ctx, &interfaces.NetOpts{
				Type:    "tcp",
				Addr:    fmt.Sprintf("%s:%s", u.Hostname(), u.Port()),
				Timeout: 200 * time.Millisecond,
			}, []byte(fmt.Sprintf("%s / HTTP/1.0\r\n\r\n", method)))
			err := worker.Do()
			if err != nil {
				t.Logf("Expected %+v; Got %+v", nil, err)
				t.FailNow()
			}
			metrics := prometheus.Metrics.Get()
			k++
			// NetWorker does not return a status code but will populate the 200 code on succesful session
			if metrics.ResponseCodes["200"] != float64(k) {
				t.Logf("Expected %+v; Got %+v", k, metrics.ResponseCodes["200"])
				t.FailNow()
			}
		})
	}
}
