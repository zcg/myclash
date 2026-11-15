package app

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo"
)

func (a *App) checkUpdates() {
	if a.updateBusy {
		return
	}
	if !mygo.Updater.Enabled() {
		a.mu.Lock()
		a.updateStatus = "当前构建未启用自动更新（开发版或未配置签名发布）"
		a.mu.Unlock()
		a.invalidate()
		return
	}
	a.mu.Lock()
	a.updateBusy = true
	a.updateStatus = "正在检查更新…"
	a.updateVersion = ""
	a.updateNotes = ""
	a.updateProgress = 0
	a.mu.Unlock()
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		up, err := mygo.Updater.Check(ctx)
		a.mu.Lock()
		a.updateBusy = false
		switch {
		case err != nil:
			a.updateStatus = "检查失败：" + err.Error()
		case up == nil:
			a.updateStatus = "已是最新版本"
		default:
			a.updateVersion = up.Version
			a.updateNotes = up.Notes
			a.updateStatus = "发现新版本 " + up.Version
		}
		a.mu.Unlock()
		a.invalidate()
	}()
}

func (a *App) installUpdate() {
	a.mu.Lock()
	busy := a.updateBusy
	a.mu.Unlock()
	if busy || !mygo.Updater.Enabled() {
		return
	}
	a.mu.Lock()
	a.updateBusy = true
	a.updateStatus = "正在下载…"
	a.updateProgress = 0
	a.mu.Unlock()
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		up, err := mygo.Updater.Check(ctx)
		if err != nil || up == nil {
			a.mu.Lock()
			a.updateBusy = false
			if err != nil {
				a.updateStatus = "检查失败：" + err.Error()
			} else {
				a.updateStatus = "已是最新版本"
			}
			a.mu.Unlock()
			a.invalidate()
			return
		}
		err = up.Install(ctx, func(downloaded, total int64) {
			if total <= 0 {
				return
			}
			a.mu.Lock()
			a.updateProgress = float64(downloaded) / float64(total)
			a.mu.Unlock()
			a.invalidate()
		})
		a.mu.Lock()
		a.updateBusy = false
		if err != nil {
			a.updateStatus = fmt.Sprintf("安装失败：%v", err)
			a.mu.Unlock()
			a.invalidate()
			return
		}
		a.updateStatus = "更新已下载，正在重启"
		a.mu.Unlock()
		a.invalidate()
		mygo.App.Relaunch()
	}()
}
