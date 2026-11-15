package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to the core's loopback REST API. It never uses a proxy: routing
// these requests through the system proxy would create a loop.
type Client struct {
	base   string
	secret string
	hc     *http.Client
	sc     *http.Client
}

func NewClient(ep Endpoint) *Client {
	tr := &http.Transport{Proxy: nil}
	return &Client{
		base:   "http://" + ep.Addr,
		secret: ep.Secret,
		hc:     &http.Client{Transport: tr, Timeout: 8 * time.Second},
		sc:     &http.Client{Transport: tr.Clone()},
	}
}

func (c *Client) Version(ctx context.Context) (Version, error) {
	var v Version
	err := c.do(ctx, http.MethodGet, "/version", nil, &v)
	return v, err
}

func (c *Client) Proxies(ctx context.Context) (map[string]Proxy, error) {
	var out struct {
		Proxies map[string]Proxy `json:"proxies"`
	}
	if err := c.do(ctx, http.MethodGet, "/proxies", nil, &out); err != nil {
		return nil, err
	}
	return out.Proxies, nil
}

func (c *Client) Select(ctx context.Context, group, name string) error {
	body := map[string]string{"name": name}
	return c.do(ctx, http.MethodPut, "/proxies/"+url.PathEscape(group), body, nil)
}

func (c *Client) Delay(ctx context.Context, name, testURL string, timeoutMS int) (int, error) {
	if testURL == "" {
		testURL = "https://www.gstatic.com/generate_204"
	}
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}
	q := url.Values{}
	q.Set("url", testURL)
	q.Set("timeout", fmt.Sprint(timeoutMS))
	var out struct {
		Delay int `json:"delay"`
	}
	err := c.do(ctx, http.MethodGet, "/proxies/"+url.PathEscape(name)+"/delay?"+q.Encode(), nil, &out)
	return out.Delay, err
}

// Traffic streams bytes-per-second samples until ctx is done or the body ends.
func (c *Client) Traffic(ctx context.Context, fn func(up, down int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/traffic", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	resp, err := c.sc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("traffic: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	dec := json.NewDecoder(resp.Body)
	for {
		var sample struct {
			Up   int64 `json:"up"`
			Down int64 `json:"down"`
		}
		if err := dec.Decode(&sample); err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		fn(sample.Up, sample.Down)
	}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Connections polls the snapshot endpoint. /connections is a normal GET on
// the Clash API, not a push stream; reading it as a stream leaves the page
// empty after the first frame.
func (c *Client) Connections(ctx context.Context, fn func(Connections)) error {
	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()
	for {
		var v Connections
		if err := c.do(ctx, http.MethodGet, "/connections", nil, &v); err != nil {
			return err
		}
		fn(v)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Logs streams log entries at the given level.
func (c *Client) Logs(ctx context.Context, level string, fn func(LogEntry)) error {
	if level == "" {
		level = "info"
	}
	body, err := c.streamGet(ctx, "/logs?level="+url.QueryEscape(level))
	if err != nil {
		return err
	}
	defer body.Close()
	dec := json.NewDecoder(body)
	for {
		var v LogEntry
		if err := dec.Decode(&v); err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		fn(v)
	}
}

func (c *Client) Rules(ctx context.Context) ([]Rule, error) {
	var out struct {
		Rules []Rule `json:"rules"`
	}
	if err := c.do(ctx, http.MethodGet, "/rules", nil, &out); err != nil {
		return nil, err
	}
	return out.Rules, nil
}

func (c *Client) CloseConnection(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/connections/"+url.PathEscape(id), nil, nil)
}

func (c *Client) CloseAllConnections(ctx context.Context) error {
	return c.do(ctx, http.MethodDelete, "/connections", nil, nil)
}

func (c *Client) streamGet(ctx context.Context, path string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	resp, err := c.sc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %s: %s", path, resp.Status, strings.TrimSpace(string(b)))
	}
	return resp.Body, nil
}

func (c *Client) Memory(ctx context.Context, fn func(inuse uint64)) error {
	body, err := c.streamGet(ctx, "/memory")
	if err != nil {
		return err
	}
	defer body.Close()
	dec := json.NewDecoder(body)
	for {
		var v struct {
			Inuse uint64 `json:"inuse"`
		}
		if err := dec.Decode(&v); err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		fn(v.Inuse)
	}
}

func (c *Client) QueryDNS(ctx context.Context, name, typ string) (DNSResponse, error) {
	if typ == "" {
		typ = "A"
	}
	q := url.Values{}
	q.Set("name", name)
	q.Set("type", typ)
	var out DNSResponse
	err := c.do(ctx, http.MethodGet, "/dns/query?"+q.Encode(), nil, &out)
	return out, err
}

func (c *Client) ClearDNSCache(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/cache/dns/flush", nil, nil)
}

func (c *Client) ClearFakeIPCache(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/cache/fakeip/flush", nil, nil)
}

func (c *Client) ProxyProviders(ctx context.Context) ([]Provider, error) {
	return c.providers(ctx, "/providers/proxies")
}

func (c *Client) RuleProviders(ctx context.Context) ([]Provider, error) {
	return c.providers(ctx, "/providers/rules")
}

func (c *Client) providers(ctx context.Context, path string) ([]Provider, error) {
	var out struct {
		Providers map[string]Provider `json:"providers"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	list := make([]Provider, 0, len(out.Providers))
	for name, p := range out.Providers {
		if p.Name == "" {
			p.Name = name
		}
		list = append(list, p)
	}
	return list, nil
}

func (c *Client) UpdateProxyProvider(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPut, "/providers/proxies/"+url.PathEscape(name), nil, nil)
}

func (c *Client) UpdateRuleProvider(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPut, "/providers/rules/"+url.PathEscape(name), nil, nil)
}

func (c *Client) HealthCheckProvider(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodGet, "/providers/proxies/"+url.PathEscape(name)+"/healthcheck", nil, nil)
}
