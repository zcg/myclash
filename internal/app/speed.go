package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"myclash/internal/core"
)

// Speed tests download a real file through the mixed port; the URL is
// configurable in Settings so users can pick a nearby mirror.

func (a *App) speedFor(name string) (float64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	mbps, ok := a.speeds[name]
	return mbps, ok
}

func (a *App) speedErrFor(name string) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	msg, ok := a.speedErrs[name]
	return msg, ok
}

func (a *App) speedFailed(name string) {
	a.win.Update(func() {
		a.mu.Lock()
		a.speedErrs[name] = "失败"
		delete(a.speeds, name)
		a.status = "测速失败：" + name
		a.mu.Unlock()
	})
}

// testSpeed temporarily selects the node in its group, downloads a fixed
// payload through the mixed port, and restores the previous selection.
func (a *App) testSpeed(group, name string) {
	a.mu.Lock()
	client := a.client
	speedURL := a.testSpeedURL
	a.mu.Unlock()
	if client == nil {
		return
	}
	port := a.settings.Get().Core.MixedPort
	go func() {
		mbps, err := a.measureSpeed(client, group, name, port, speedURL)
		if err != nil {
			a.speedFailed(name)
			return
		}
		a.win.Update(func() {
			a.mu.Lock()
			a.speeds[name] = mbps
			delete(a.speedErrs, name)
			a.status = fmt.Sprintf("%s 测速 %.1f Mbps", name, mbps)
			a.mu.Unlock()
		})
	}()
}

func (a *App) testGroupSpeed(group string) {
	a.mu.Lock()
	client := a.client
	speedURL := a.testSpeedURL
	a.mu.Unlock()
	if client == nil {
		return
	}
	port := a.settings.Get().Core.MixedPort
	proxies, err := client.Proxies(context.Background())
	if err != nil {
		a.setError(err)
		return
	}
	members := proxies[group].All
	go func() {
		for _, member := range members {
			mbps, err := a.measureSpeed(client, group, member, port, speedURL)
			if err != nil {
				a.speedFailed(member)
				continue
			}
			a.win.Update(func() {
				a.mu.Lock()
				a.speeds[member] = mbps
				delete(a.speedErrs, member)
				a.mu.Unlock()
			})
		}
		a.win.Update(func() {
			a.mu.Lock()
			a.status = group + " 测速完成"
			a.mu.Unlock()
		})
	}()
}

func (a *App) measureSpeed(client *core.Client, group, name string, port int, speedURL string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	proxies, err := client.Proxies(ctx)
	if err != nil {
		return 0, err
	}
	previous := proxies[group].Now
	if err := client.Select(ctx, group, name); err != nil {
		return 0, err
	}
	defer func() {
		if previous != "" && previous != name {
			_ = client.Select(context.Background(), group, previous)
		}
	}()

	proxyURL := &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", port)}
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true}
	defer tr.CloseIdleConnections()
	httpClient := &http.Client{Transport: tr, Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, speedURL, nil)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, err
	}
	secs := time.Since(start).Seconds()
	if secs <= 0 {
		return 0, nil
	}
	return float64(n) * 8 / secs / 1e6, nil
}
