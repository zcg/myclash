package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"myclash/internal/ui/acrylic"
)

func (a *App) openProfileEditor(uid string) {
	raw, err := a.profiles.Raw(uid)
	if err != nil {
		a.setError(err)
		return
	}
	buf := editor.NewBuffer(string(raw))
	buf.SetLanguage(editor.LanguageFor("profile.yaml"))
	a.baseEdit = &editor.State{Buffer: buf}
	a.baseUID = uid
}

func (a *App) pageProfileEditor(c *ui.Context, p acrylic.Palette) {
	meta, ok := a.profiles.CurrentMeta()
	name := a.baseUID
	if ok && meta.UID == a.baseUID {
		name = meta.Name
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		if acrylic.Button(c, p, "返回", false).Clicked() {
			a.baseUID = ""
			a.baseEdit = nil
		}
		ui.Column(c).Gap(2).Children(func() {
			ui.Textf(c, "编辑 %s", name).FontSize(22).Bold()
			ui.Text(c, "保存后校验并重新应用；失败保留运行中配置").FontSize(12).TextColor(p.InkSubtle)
		})
		ui.Spacer(c)
		if acrylic.PrimaryButton(c, p, "保存并应用", false).Clicked() {
			a.saveBaseProfile()
		}
	})
	a.errorBar(c, p)

	if a.baseEdit == nil {
		ui.Text(c, "没有可编辑的内容。").TextColor(p.InkMuted)
		return
	}
	ui.Column(c).Height(560).Background(p.Code).Radius(8).Padding(10).Clip().Children(func() {
		editor.View(c, a.baseEdit).Fill()
	})
}

func (a *App) saveBaseProfile() {
	if a.baseEdit == nil || a.baseUID == "" {
		return
	}
	raw := a.baseEdit.Buffer.String()
	if err := a.profiles.SaveRaw(a.baseUID, []byte(raw)); err != nil {
		a.setError(err)
		return
	}
	uid := a.baseUID
	a.baseUID = ""
	a.baseEdit = nil
	if a.profiles.Selected() == uid && a.state.Get().Started {
		a.restartCore()
	}
	a.invalidate()
}
