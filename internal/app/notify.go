package app

import "github.com/egoist/mygo"

func (a *App) notify(title, body string) {
	if !a.notifications || !mygo.NotificationsSupported() {
		return
	}
	// In-window results already use the error bar/status strip; only notify
	// when the user cannot see the window.
	if a.win != nil && a.win.IsVisible() {
		return
	}
	n := mygo.NewNotification(mygo.NotificationOptions{
		ID:    "myclash:" + title,
		Title: title,
		Body:  body,
	})
	_ = n.Show()
}
