package app

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/egoist/mygo"

	"myclash/internal/platform"
)

//go:embed tray.png
var trayIcon []byte

func (a *App) setupTray() {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:      "main",
		Icon:    trayIcon,
		ToolTip: "myclash",
		Menu:    a.trayMenu(),
	})
	if err != nil {
		a.setError(fmt.Errorf("托盘初始化失败: %w", err))
		return
	}
	a.tray = tray
	a.tray.OnClick(a.showMainWindow)
}

func (a *App) showMainWindow() {
	if a.win == nil {
		return
	}
	a.win.Show()
	a.win.Restore()
	a.win.Focus()
}

func (a *App) refreshTrayMenu() {
	if a.tray == nil {
		return
	}
	a.tray.SetMenu(a.trayMenu())
}

func (a *App) trayMenu() *mygo.Menu {
	mode := a.settings.Get().Core.Mode
	radio := func(label, value string) *mygo.MenuItem {
		return &mygo.MenuItem{
			Label:   label,
			Type:    mygo.MenuItemRadio,
			Checked: mode == value,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.modeIdx = modeIndex(value)
				a.applyMode(a.modeIdx)
				a.refreshTrayMenu()
			},
		}
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "显示窗口", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.showMainWindow()
		}},
		{Label: "悬浮窗", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.toggleFloating()
		}},
		{Type: mygo.MenuItemSeparator},
		{Label: a.profileName(), Disabled: true},
		{Type: mygo.MenuItemSeparator},
		radio("规则模式", "rule"),
		radio("全局模式", "global"),
		radio("直连模式", "direct"),
		{Type: mygo.MenuItemSeparator},
		{Label: "TUN 模式", Type: mygo.MenuItemCheckbox, Checked: a.tunEnable, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.tunEnable = !a.tunEnable
			a.toggleTUN()
			a.refreshTrayMenu()
		}},
		{Label: "重启内核", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.restartCore()
		}},
		{Label: "系统代理", Type: mygo.MenuItemCheckbox, Checked: a.systemProxy, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.systemProxy = !a.systemProxy
			a.toggleSystemProxy(a.systemProxy)
			a.refreshTrayMenu()
		}},
		{Type: mygo.MenuItemSeparator},
		{Label: "退出 myclash", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.quitApp()
		}},
	})
}

func (a *App) profileName() string {
	if meta, ok := a.profiles.CurrentMeta(); ok && meta.Name != "" {
		return "配置：" + meta.Name
	}
	return "未启用配置"
}

func (a *App) registerHotkeys() {
	mygo.GlobalShortcut.UnregisterAll()
	if a.win == nil {
		return
	}
	register := func(acc string, fn func()) {
		if acc == "" {
			return
		}
		_ = mygo.GlobalShortcut.Register(acc, fn)
	}
	register(a.hotkeyShow, func() {
		a.win.Show()
		a.win.Restore()
		a.win.Focus()
	})
	register(a.hotkeyProxy, func() {
		a.systemProxy = !a.systemProxy
		a.toggleSystemProxy(a.systemProxy)
		a.refreshTrayMenu()
	})
	register(a.hotkeyTUN, func() {
		a.tunEnable = !a.tunEnable
		a.toggleTUN()
		a.refreshTrayMenu()
	})
	register(a.hotkeyMode, func() {
		a.modeIdx = (a.modeIdx + 1) % 3
		a.applyMode(a.modeIdx)
		a.refreshTrayMenu()
	})
	register(a.hotkeyQuit, a.quitApp)
	// Quitting must not depend on a profile that may have cleared its hotkeys.
	register("CmdOrCtrl+Shift+Q", a.quitApp)
	register("CmdOrCtrl+K", func() {
		a.paletteOpen = !a.paletteOpen
		a.paletteQuery = ""
		a.paletteIndex = 0
		if a.win != nil {
			a.win.Show()
			a.win.Focus()
		}
		a.invalidate()
	})
}

func (a *App) quitApp() {
	a.forceQuit = true
	// Remove the notification icon first; the window and its menu callback can
	// be the very thing that is stuck.
	if a.tray != nil {
		a.tray.Destroy()
		a.tray = nil
	}
	platform.ArmKillWatchdog()
	// Stop the embedded core so Wintun and its routes are released, but never
	// let a stuck shutdown keep the process alive. App.Quit depends on the
	// native event loop finishing; in this path that is exactly what is broken.
	go func() {
		a.flushTrafficHistory()
		done := make(chan struct{})
		go func() {
			a.stopCore()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(1500 * time.Millisecond):
		}
		os.Exit(0)
	}()
}
