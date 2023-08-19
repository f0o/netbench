package worker

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/goleak"
	"golang.org/x/net/nettest"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/prometheus"
)

func init() {
	prometheus.SkipSanityCheck = true
}

func TestHTTPWorker(t *testing.T) {
	defer goleak.VerifyNone(t)
	for k, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				_, err := w.Write([]byte("Hello World"))
				if err != nil {
					t.Logf("Cannot write reply")
					t.FailNow()
				}
				if r.Method != method {
					t.Logf("Expected %+v; Got %+v", method, r.Method)
					t.FailNow()
					return
				}
				body, _ := io.ReadAll(r.Body)
				if string(body) != "Hello World" {
					t.Logf("Expected %+v; Got %+v", "Hello World", string(body))
					t.FailNow()
					return
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
				return
			}
			metrics := prometheus.Metrics.Get()
			k++
			if metrics.ResponseCodes["201"] != float64(k) {
				t.Logf("Expected %+v; Got %+v", k, metrics.ResponseCodes["201"])
				t.FailNow()
				return
			}
		})
	}
}

func TestNetWorker(t *testing.T) {
	defer goleak.VerifyNone(t)
	inc := 0.0
	for _, network := range []string{"tcp", "tcp4", "tcp6", "unix", "unixpacket"} {
		t.Run(network, func(t *testing.T) {
			payload := make([]byte, 16)
			_, err := rand.Read(payload)
			if err != nil {
				t.Logf("Cannot capture random payload")
				t.SkipNow()
			}

			if nettest.TestableNetwork(network) == false {
				t.SkipNow()
				return
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ts, err := nettest.NewLocalListener(network)
			if err != nil {
				t.Logf("Nettest Error; %+v", err)
				t.FailNow()
				return
			}
			go func() {
				c, err := ts.Accept()
				if err != nil {
					t.Logf("Nettest Accept Error; %+v", err)
					t.Fail()
					return
				}
				defer c.Close()
				buf := make([]byte, 16)
				n, err := c.Read(buf)
				if err != nil {
					t.Logf("Nettest Read Error; %+v", err)
					t.Fail()
					return
				}
				if string(buf[:n]) != string(payload) {
					t.Logf("Expected %x; Got %x", string(payload), string(buf[:n]))
					t.Fail()
					return
				}
			}()
			defer ts.Close()
			worker := NewNetWorker(ctx, &interfaces.NetOpts{
				Type:    network,
				Addr:    ts.Addr().String(),
				Timeout: 200 * time.Millisecond,
			}, payload)
			err = worker.Do()
			if err != nil {
				t.Logf("Expected %+v; Got %+v", nil, err)
				t.FailNow()
			}
			metrics := prometheus.Metrics.Get()
			inc++
			// NetWorker does not return a status code but will populate the 200 code on succesful session
			if metrics.ResponseCodes["200"] != inc {
				t.Logf("Expected %+v; Got %+v", inc, metrics.ResponseCodes["200"])
				t.FailNow()
				return
			}
		})
	}
}
