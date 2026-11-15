package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientVersionAndProxies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing bearer token: %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/version":
			fmt.Fprint(w, `{"meta":true,"version":"v1.19.34"}`)
		case "/proxies":
			fmt.Fprint(w, `{"proxies":{"G":{"name":"G","type":"Selector","now":"A","all":["A","B"]}}}`)
		case "/proxies/G/delay":
			if r.URL.Query().Get("url") == "" || r.URL.Query().Get("timeout") == "" {
				t.Errorf("missing delay query: %s", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"delay":42}`)
		case "/proxies/G":
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			fmt.Fprint(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Endpoint{Addr: srv.Listener.Addr().String(), Secret: "secret"})
	v, err := c.Version(context.Background())
	if err != nil || v.Version != "v1.19.34" {
		t.Fatalf("version = %+v, %v", v, err)
	}
	all, err := c.Proxies(context.Background())
	if err != nil || len(all["G"].All) != 2 {
		t.Fatalf("proxies = %+v, %v", all, err)
	}
	ms, err := c.Delay(context.Background(), "G", "", 5000)
	if err != nil || ms != 42 {
		t.Fatalf("delay = %d, %v", ms, err)
	}
	if err := c.Select(context.Background(), "G", "B"); err != nil {
		t.Fatal(err)
	}
}

func TestClientTraffic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"up":1,"down":2}`)
		fmt.Fprintln(w, `{"up":3,"down":4}`)
	}))
	defer srv.Close()

	c := NewClient(Endpoint{Addr: srv.Listener.Addr().String(), Secret: "secret"})
	var ups, downs int64
	err := c.Traffic(context.Background(), func(up, down int64) {
		ups += up
		downs += down
	})
	if err != nil {
		t.Fatal(err)
	}
	if ups != 4 || downs != 6 {
		t.Fatalf("totals = %d/%d, want 4/6", ups, downs)
	}
}
