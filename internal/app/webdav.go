package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"myclash/internal/settings"
)

func (a *App) saveWebDAV() {
	if err := a.settings.Update(func(s *settings.Settings) {
		s.WebDAV.URL = strings.TrimSpace(a.webdavURL)
		s.WebDAV.Username = strings.TrimSpace(a.webdavUser)
		s.WebDAV.Password = a.webdavPass
	}); err != nil {
		a.setError(err)
		return
	}
	a.setStatus("WebDAV 设置已保存")
}

func (a *App) webdavEndpoint() (string, error) {
	base := strings.TrimSpace(a.webdavURL)
	if base == "" {
		return "", errors.New("先填写 WebDAV 地址")
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("WebDAV 地址无效")
	}
	if strings.HasSuffix(u.Path, "/") || u.Path == "" {
		u.Path = path.Join(u.Path, "myclash-backup.zip")
	}
	return u.String(), nil
}

func (a *App) webdavClient() (*http.Client, *http.Transport) {
	tr := &http.Transport{Proxy: nil}
	return &http.Client{Timeout: 45 * time.Second, Transport: tr}, tr
}

func (a *App) uploadWebDAV() {
	endpoint, err := a.webdavEndpoint()
	if err != nil {
		a.setError(err)
		return
	}
	tmp, err := os.CreateTemp("", "myclash-webdav-*.zip")
	if err != nil {
		a.setError(err)
		return
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	if err := writeZip(name, a.dataDir, true); err != nil {
		a.setError(err)
		return
	}
	f, err := os.Open(name)
	if err != nil {
		a.setError(err)
		return
	}
	defer f.Close()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, endpoint, f)
	if err != nil {
		a.setError(err)
		return
	}
	if a.webdavUser != "" {
		req.SetBasicAuth(a.webdavUser, a.webdavPass)
	}
	client, tr := a.webdavClient()
	defer tr.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		a.setError(fmt.Errorf("WebDAV 上传失败: %w", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		a.setError(fmt.Errorf("WebDAV 上传失败: HTTP %d", resp.StatusCode))
		return
	}
	a.setStatus("备份已上传到 WebDAV")
}

func (a *App) downloadWebDAV() {
	endpoint, err := a.webdavEndpoint()
	if err != nil {
		a.setError(err)
		return
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		a.setError(err)
		return
	}
	if a.webdavUser != "" {
		req.SetBasicAuth(a.webdavUser, a.webdavPass)
	}
	client, tr := a.webdavClient()
	defer tr.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		a.setError(fmt.Errorf("WebDAV 下载失败: %w", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		a.setError(fmt.Errorf("WebDAV 下载失败: HTTP %d", resp.StatusCode))
		return
	}
	tmp, err := os.CreateTemp("", "myclash-webdav-*.zip")
	if err != nil {
		a.setError(err)
		return
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, 256<<20)); err != nil {
		tmp.Close()
		a.setError(err)
		return
	}
	tmp.Close()
	if err := unzipInto(name, a.dataDir); err != nil {
		a.setError(err)
		return
	}
	a.setStatus("已从 WebDAV 恢复，重启应用生效")
	a.requestRelaunch()
}
