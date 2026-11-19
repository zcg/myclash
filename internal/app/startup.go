package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/egoist/mygo"
)

// startupHealthCheck verifies that the fresh instance owns its listener and
// can actually reach the internet through the embedded core. A stale process
// from an interrupted update can keep the mixed port while pointing the
// system proxy at itself; warn instead of leaving the user without a network.
func (a *App) startupHealthCheck() {
	go func() {
		// Give the core, system proxy and TUN a moment to settle after launch.
		time.Sleep(3 * time.Second)

		// Only check when a core is expected. A stopped core is a normal
		// state, not a stale process or a broken network.
		if !a.state.View().Started {
			return
		}

		port := a.settings.Get().Core.MixedPort
		addr := net.JoinHostPort("127.0.0.1", fmt.Sprint(port))
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			a.alertNetworkProblem("代理端口未监听",
				"端口 "+fmt.Sprint(port)+" 无法连接，可能有僵死进程占用。请检查任务管理器中的 myclash 进程。")
			return
		}
		_ = conn.Close()

		proxyURL := &url.URL{Scheme: "http", Host: addr}
		tr := &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true}
		defer tr.CloseIdleConnections()
		client := &http.Client{Transport: tr, Timeout: 8 * time.Second}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
			"https://www.gstatic.com/generate_204", nil)
		if err == nil {
			resp, doErr := client.Do(req)
			if doErr == nil {
				_ = doErr
				_ = resp.Body.Close()
			}
			err = doErr
		}
		if err != nil {
			a.alertNetworkProblem("网络自检失败",
				"通过本地代理无法访问外网，请检查是否有僵死进程占用端口或代理节点是否可用。")
		}
	}()
}

func (a *App) alertNetworkProblem(title, body string) {
	if !mygo.NotificationsSupported() {
		return
	}
	n := mygo.NewNotification(mygo.NotificationOptions{
		ID:     "myclash:network-check",
		Title:  title,
		Body:   body,
		Silent: false,
	})
	_ = n.Show()
	a.updateUI(func() {
		a.mu.Lock()
		a.errMsg = body
		a.mu.Unlock()
		a.invalidate()
	})
}
