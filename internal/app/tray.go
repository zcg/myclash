package app

import (
	_ "embed"

	"github.com/egoist/mygo"
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
		return
	}
	a.tray = tray
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
			if a.win != nil {
				a.win.Show()
			}
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
			a.forceQuit = true
			mygo.App.Quit()
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
}
