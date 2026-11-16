package app

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/egoist/mygo"
)

type trayAgent struct {
	tray *mygo.Tray
}

func runTrayAgent() error {
	a := &trayAgent{}
	mygo.App.WhenReady(a.ready)
	return mygo.App.Run()
}

func (a *trayAgent) ready() {
	if !agentAlive() {
		exe, err := os.Executable()
		if err != nil {
			return
		}
		cmd := exec.Command(exe, "--core")
		if err := cmd.Start(); err != nil {
			return
		}
		_ = cmd.Process.Release()
		// Give the headless core a moment to write its endpoint.
		for i := 0; i < 60 && !agentAlive(); i++ {
			time.Sleep(100 * time.Millisecond)
		}
	}
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID: "agent", Icon: trayIcon, ToolTip: "myclash Agent",
		Menu: mygo.NewMenu([]*mygo.MenuItem{
			{Label: "显示窗口", Click: func(*mygo.MenuItem, *mygo.Window) {
				exe, err := os.Executable()
				if err != nil {
					return
				}
				cmd := exec.Command(exe, "--attach")
				_ = cmd.Start()
			}},
			{Type: mygo.MenuItemSeparator},
			{Label: "重启内核", Click: func(*mygo.MenuItem, *mygo.Window) {
				_ = a.post("/restart")
			}},
			{Label: "退出 Agent", Click: func(*mygo.MenuItem, *mygo.Window) {
				_ = a.post("/quit")
				if a.tray != nil {
					a.tray.Destroy()
					a.tray = nil
				}
				go func() {
					time.Sleep(100 * time.Millisecond)
					os.Exit(0)
				}()
			}},
		}),
	})
	if err == nil {
		a.tray = tray
	}
}

func (a *trayAgent) post(path string) error {
	info, err := readAgentInfo()
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+info.ControlAddr+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+info.ControlToken)
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("agent: %s", resp.Status)
	}
	return nil
}
