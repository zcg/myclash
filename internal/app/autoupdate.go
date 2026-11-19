package app

import (
	"context"
	"math/rand"
	"time"

	"github.com/egoist/mygo"

	"myclash/internal/settings"
)

const (
	autoUpdateInterval   = 24 * time.Hour
	autoUpdateJitterMax  = 6 * time.Hour
	autoUpdateStartupMin = 30 * time.Second
)

// startAutoUpdateCheck schedules a daily automatic update check. The first
// check is delayed a little after launch so the window is ready, and carries
// random jitter so a fleet of installs does not all hit the update server at
// the same moment. A "Cancel" answer disables the feature permanently.
func (a *App) startAutoUpdateCheck() {
	if !a.settings.AutoCheckUpdates() || !mygo.Updater.Enabled() {
		return
	}
	last := a.settings.Get().App.LastUpdateCheck
	wait := autoUpdateInterval - time.Since(last)
	if wait > autoUpdateInterval || wait < 0 {
		wait = autoUpdateInterval
	}
	if last.IsZero() {
		wait = autoUpdateStartupMin
	}
	jitter := time.Duration(rand.Int63n(int64(autoUpdateJitterMax)))
	wait += jitter

	ctx, cancel := context.WithCancel(context.Background())
	a.autoUpdateCancel = cancel
	go func() {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}

			if !a.settings.AutoCheckUpdates() {
				return
			}
			a.autoCheckOnce()

			// Persist the check time so a restart does not immediately check
			// again, and schedule the next daily pass with fresh jitter.
			_ = a.settings.Update(func(s *settings.Settings) {
				s.App.LastUpdateCheck = time.Now()
			})
			next := autoUpdateInterval + time.Duration(rand.Int63n(int64(autoUpdateJitterMax)))
			timer.Reset(next)
		}
	}()
}

func (a *App) autoCheckOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	up, err := mygo.Updater.Check(ctx)
	if err != nil || up == nil {
		return
	}

	// Dialog.Message must run on the UI thread.
	a.updateUI(func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:        a.win,
			Type:          mygo.MessageInfo,
			Title:         "发现新版本",
			Message:       "myclash " + up.Version + " 已发布，是否现在更新？",
			Detail:        "选择取消后将不再自动检查更新，你仍可在设置中手动检查。",
			Buttons:       []string{"更新", "取消"},
			DefaultButton: 0,
			CancelButton:  1,
		})
		if err != nil || res.Button == 1 {
			// Cancel (or a dismissed dialog): stop bothering the user.
			_ = a.settings.SetAutoCheckUpdates(false)
			return
		}
		// Update: reuse the existing install-and-restart flow. Populate the
		// shared update state so the settings page reflects progress too.
		a.mu.Lock()
		a.updateBusy = false
		a.updateVersion = up.Version
		a.updateNotes = up.Notes
		a.mu.Unlock()
		a.installUpdate()
	})
}
