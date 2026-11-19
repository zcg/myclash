package config

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Userinfo struct {
	Upload   int64 `yaml:"upload" json:"upload"`
	Download int64 `yaml:"download" json:"download"`
	Total    int64 `yaml:"total" json:"total"`
	Expire   int64 `yaml:"expire" json:"expire"`
}

type Profile struct {
	UID       string    `yaml:"uid"`
	Name      string    `yaml:"name"`
	Source    string    `yaml:"source"` // subscription | local
	URL       string    `yaml:"url,omitempty"`
	UpdatedAt time.Time `yaml:"updated_at"`
	Userinfo  *Userinfo `yaml:"userinfo,omitempty"`
	Raw       []byte    `yaml:"-"`
}

type index struct {
	Schema   int       `yaml:"schema"`
	Selected string    `yaml:"selected"`
	Items    []Profile `yaml:"items"`
}

// Store owns profiles.yaml and the raw profile files.
type Store struct {
	mu      sync.RWMutex
	dir     string
	index   string
	profile string
	data    index
}

func NewStore(userData string) (*Store, error) {
	if userData == "" {
		return nil, errors.New("config: empty user data directory")
	}
	s := &Store{
		dir:     userData,
		index:   filepath.Join(userData, "profiles.yaml"),
		profile: filepath.Join(userData, "profiles"),
		data:    index{Schema: 1},
	}
	if err := os.MkdirAll(s.profile, 0o700); err != nil {
		return nil, fmt.Errorf("config: create profiles dir: %w", err)
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.index)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("config: read index: %w", err)
	}
	var v index
	if err := yaml.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("config: parse index: %w", err)
	}
	s.data = v
	return nil
}

func (s *Store) List() []Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Profile, len(s.data.Items))
	copy(out, s.data.Items)
	return out
}

func (s *Store) Selected() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Selected
}

func (s *Store) CurrentMeta() (Profile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.data.Items {
		if p.UID == s.data.Selected {
			return p, true
		}
	}
	return Profile{}, false
}

func (s *Store) Current() (Profile, bool) {
	p, ok := s.CurrentMeta()
	if !ok {
		return Profile{}, false
	}
	p.Raw, _ = os.ReadFile(s.pathFor(p.UID))
	return p, true
}

func (s *Store) AddLocal(name string, raw []byte) (Profile, error) {
	if len(raw) == 0 {
		return Profile{}, errors.New("config: empty profile")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "本地配置"
	}
	uid, err := newUID()
	if err != nil {
		return Profile{}, err
	}
	p := Profile{UID: uid, Name: name, Source: "local", UpdatedAt: time.Now()}
	if err := os.WriteFile(s.pathFor(uid), raw, 0o600); err != nil {
		return Profile{}, fmt.Errorf("config: write profile: %w", err)
	}
	s.mu.Lock()
	s.data.Items = append(s.data.Items, p)
	if s.data.Selected == "" {
		s.data.Selected = uid
	}
	err = s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s *Store) AddSubscription(ctx context.Context, name, rawURL string) (Profile, error) {
	rawURL = strings.TrimSpace(rawURL)
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Profile{}, errors.New("config: subscription URL must be http(s)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Profile{}, err
	}
	req.Header.Set("User-Agent", "clash-verge/v2.4.2")
	tr := &http.Transport{Proxy: nil}
	defer tr.CloseIdleConnections()
	hc := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}
	resp, err := hc.Do(req)
	if err != nil {
		return Profile{}, fmt.Errorf("config: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Profile{}, fmt.Errorf("config: download: %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return Profile{}, fmt.Errorf("config: read subscription: %w", err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = u.Hostname()
	}
	uid, err := newUID()
	if err != nil {
		return Profile{}, err
	}
	p := Profile{
		UID:       uid,
		Name:      name,
		Source:    "subscription",
		URL:       rawURL,
		UpdatedAt: time.Now(),
		Userinfo:  parseUserinfo(resp.Header.Get("subscription-userinfo")),
	}
	if err := os.WriteFile(s.pathFor(uid), raw, 0o600); err != nil {
		return Profile{}, fmt.Errorf("config: write profile: %w", err)
	}
	s.mu.Lock()
	s.data.Items = append(s.data.Items, p)
	if s.data.Selected == "" {
		s.data.Selected = uid
	}
	err = s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s *Store) Select(uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.data.Items {
		if p.UID == uid {
			s.data.Selected = uid
			return s.saveLocked()
		}
	}
	return fmt.Errorf("config: profile %q not found", uid)
}

func (s *Store) Delete(uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Selected == uid {
		return errors.New("config: cannot delete the selected profile")
	}
	for i, p := range s.data.Items {
		if p.UID == uid {
			s.data.Items = append(s.data.Items[:i], s.data.Items[i+1:]...)
			if err := s.saveLocked(); err != nil {
				return err
			}
			return os.Remove(s.pathFor(uid))
		}
	}
	return fmt.Errorf("config: profile %q not found", uid)
}

func (s *Store) pathFor(uid string) string { return filepath.Join(s.profile, uid+".yaml") }

func (s *Store) saveLocked() error {
	data, err := yaml.Marshal(s.data)
	if err != nil {
		return err
	}
	tmp := s.index + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.index)
}

func newUID() (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "p_" + hex.EncodeToString(b[:]), nil
}

func parseUserinfo(v string) *Userinfo {
	if v == "" {
		return nil
	}
	u := &Userinfo{}
	for _, part := range strings.Split(v, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		var n int64
		if _, err := fmt.Sscan(kv[1], &n); err != nil {
			continue
		}
		switch kv[0] {
		case "upload":
			u.Upload = n
		case "download":
			u.Download = n
		case "total":
			u.Total = n
		case "expire":
			u.Expire = n
		}
	}
	return u
}

// Sync re-downloads a subscription in place, keeping its UID and selection.
func (s *Store) Sync(ctx context.Context, uid string) (Profile, error) {
	s.mu.RLock()
	var target Profile
	found := false
	for _, p := range s.data.Items {
		if p.UID == uid {
			target, found = p, true
			break
		}
	}
	s.mu.RUnlock()
	if !found {
		return Profile{}, fmt.Errorf("config: profile %q not found", uid)
	}
	if target.URL == "" {
		return Profile{}, errors.New("config: profile has no subscription URL")
	}
	raw, info, err := fetchSubscription(ctx, target.URL)
	if err != nil {
		return Profile{}, err
	}
	if err := os.WriteFile(s.pathFor(uid), raw, 0o600); err != nil {
		return Profile{}, fmt.Errorf("config: write profile: %w", err)
	}
	s.mu.Lock()
	for i := range s.data.Items {
		if s.data.Items[i].UID == uid {
			s.data.Items[i].UpdatedAt = time.Now()
			s.data.Items[i].Userinfo = info
			target = s.data.Items[i]
			break
		}
	}
	err = s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return Profile{}, err
	}
	return target, nil
}

func fetchSubscription(ctx context.Context, rawURL string) ([]byte, *Userinfo, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, nil, errors.New("config: subscription URL must be http(s)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "clash-verge/v2.4.2")
	tr := &http.Transport{Proxy: nil}
	defer tr.CloseIdleConnections()
	hc := &http.Client{Timeout: 30 * time.Second, Transport: tr}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("config: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("config: download: %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("config: read subscription: %w", err)
	}
	return raw, parseUserinfo(resp.Header.Get("subscription-userinfo")), nil
}

func (s *Store) overridePath(uid string) string {
	return filepath.Join(s.dir, "overrides", uid+".yaml")
}

func (s *Store) Override(uid string) ([]byte, error) {
	data, err := os.ReadFile(s.overridePath(uid))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

func (s *Store) SaveOverride(uid string, text []byte) error {
	path := s.overridePath(uid)
	if len(text) == 0 {
		err := os.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, text, 0o600)
}

func (s *Store) Rename(uid, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Items {
		if s.data.Items[i].UID == uid {
			s.data.Items[i].Name = name
			return s.saveLocked()
		}
	}
	return fmt.Errorf("config: profile %q not found", uid)
}

func (s *Store) Duplicate(uid string) (Profile, error) {
	s.mu.RLock()
	var src Profile
	found := false
	for _, p := range s.data.Items {
		if p.UID == uid {
			src, found = p, true
			break
		}
	}
	s.mu.RUnlock()
	if !found {
		return Profile{}, fmt.Errorf("config: profile %q not found", uid)
	}
	raw, err := os.ReadFile(s.pathFor(uid))
	if err != nil {
		return Profile{}, err
	}
	return s.AddLocal(src.Name+" 副本", raw)
}

func (s *Store) SaveRaw(uid string, raw []byte) error {
	if len(raw) == 0 {
		return errors.New("config: empty profile")
	}
	if err := os.WriteFile(s.pathFor(uid), raw, 0o600); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Items {
		if s.data.Items[i].UID == uid {
			s.data.Items[i].UpdatedAt = time.Now()
			return s.saveLocked()
		}
	}
	return fmt.Errorf("config: profile %q not found", uid)
}
func (s *Store) Raw(uid string) ([]byte, error) {
	return os.ReadFile(s.pathFor(uid))
}

func (s *Store) scriptPath(uid string) string {
	return filepath.Join(s.dir, "overrides", uid+".js")
}

func (s *Store) ScriptOverride(uid string) ([]byte, error) {
	if uid == "" {
		return nil, nil
	}
	b, err := os.ReadFile(s.scriptPath(uid))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

func (s *Store) SaveScriptOverride(uid, script string) error {
	if uid == "" {
		return errors.New("config: empty profile uid")
	}
	path := s.scriptPath(uid)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(script), 0o600)
}
