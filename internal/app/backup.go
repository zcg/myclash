package app

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

func (a *App) backupData() {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      a.win,
		Title:       "导出 myclash 备份",
		DefaultPath: "myclash-backup.zip",
		Filters:     []mygo.FileFilter{{Name: "ZIP", Extensions: []string{"zip"}}},
	})
	if err != nil || path == "" {
		return
	}
	if err := writeZip(path, a.dataDir, true); err != nil {
		a.setError(err)
		return
	}
	a.setStatus("备份已保存：" + path)
}

func (a *App) restoreData() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  a.win,
		Title:   "选择 myclash 备份",
		Filters: []mygo.FileFilter{{Name: "ZIP", Extensions: []string{"zip"}}},
	})
	if err != nil || len(paths) == 0 {
		return
	}
	if err := unzipInto(paths[0], a.dataDir); err != nil {
		a.setError(err)
		return
	}
	a.setStatus("备份已恢复，重启应用生效")
	a.requestRelaunch()
}

func (a *App) exportDiagnostics() {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      a.win,
		Title:       "导出诊断包",
		DefaultPath: "myclash-diagnostics.zip",
		Filters:     []mygo.FileFilter{{Name: "ZIP", Extensions: []string{"zip"}}},
	})
	if err != nil || path == "" {
		return
	}
	if err := writeZip(path, a.dataDir, false); err != nil {
		a.setError(err)
		return
	}
	a.setStatus("诊断包已保存：" + path)
}

func (a *App) openDataDir() {
	if err := mygo.Shell.OpenPath(a.dataDir); err != nil {
		a.setError(err)
	}
}

func (a *App) resetData() {
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:  a.win,
		Type:    mygo.MessageWarning,
		Title:   "重置 myclash",
		Message: "删除所有配置、订阅和设置？",
		Detail:  "此操作不可撤销。备份文件不会被删除。",
		Buttons: []string{"取消", "重置"},
	})
	if err != nil || res.Button != 1 {
		return
	}
	_ = os.RemoveAll(a.dataDir)
	a.requestRelaunch()
}

func writeZip(dest, dataDir string, includeProfiles bool) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	files := []string{"settings.yaml", "profiles.yaml", "window-state.json"}
	if includeProfiles {
		files = append(files, "logs/core.log", "overrides", "profiles")
	}
	for _, rel := range files {
		p := filepath.Join(dataDir, rel)
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.IsDir() {
			err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}
				return zipOne(zw, dataDir, path)
			})
			if err != nil {
				return err
			}
			continue
		}
		if err := zipOne(zw, dataDir, p); err != nil {
			return err
		}
	}
	return nil
}

func zipOne(zw *zip.Writer, root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	w, err := zw.Create(filepath.ToSlash(rel))
	if err != nil {
		return err
	}
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(w, in)
	return err
}

func unzipInto(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		path := filepath.Join(dst, filepath.FromSlash(f.Name))
		if !filepath.IsLocal(filepath.FromSlash(f.Name)) {
			return fmt.Errorf("备份包含非法路径 %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
