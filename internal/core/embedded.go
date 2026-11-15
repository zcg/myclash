package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	mihomoconfig "github.com/metacubex/mihomo/config"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/hub/route"
	"github.com/metacubex/mihomo/tunnel"
)

// Embedded hosts mihomo in the current process. Only one Embedded may run per
// process at a time: mihomo keeps process-wide state.
type Embedded struct {
	mu       sync.Mutex
	home     string
	endpoint Endpoint
	started  bool
}

func NewEmbedded(home string, controllerPort int) (*Embedded, error) {
	if home == "" {
		return nil, errors.New("core: empty home directory")
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, fmt.Errorf("core: create home %s: %w", home, err)
	}
	addr, err := controllerAddr(controllerPort)
	if err != nil {
		return nil, err
	}
	secret, err := randomSecret()
	if err != nil {
		return nil, err
	}
	return &Embedded{
		home:     home,
		endpoint: Endpoint{Addr: addr, Secret: secret},
	}, nil
}

func (e *Embedded) Start(ctx context.Context, cfg []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return e.reloadLocked(cfg)
	}
	C.SetHomeDir(e.home)
	if err := mihomoconfig.Init(e.home); err != nil {
		return fmt.Errorf("core: init home %s: %w", e.home, err)
	}
	// Disable /restart, /upgrade and config writes: an app embedding mihomo
	// must not let a local page replace its binary or restart the GUI.
	route.SetEmbedMode(true)
	if err := hub.Parse(cfg,
		hub.WithExternalController(e.endpoint.Addr),
		hub.WithSecret(e.endpoint.Secret),
	); err != nil {
		return fmt.Errorf("core: start: %w", err)
	}
	e.started = true
	return nil
}

func (e *Embedded) Reload(ctx context.Context, cfg []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.started {
		return errors.New("core: not started")
	}
	return e.reloadLocked(cfg)
}

func (e *Embedded) reloadLocked(cfg []byte) error {
	parsed, err := executor.ParseWithBytes(cfg)
	if err != nil {
		return fmt.Errorf("core: parse config: %w", err)
	}
	executor.ApplyConfig(parsed, false)
	return nil
}

func (e *Embedded) SetMode(ctx context.Context, mode string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.started {
		return errors.New("core: not started")
	}
	m, ok := tunnel.ModeMapping[mode]
	if !ok {
		return fmt.Errorf("core: unknown mode %q", mode)
	}
	tunnel.SetMode(m)
	return nil
}

func (e *Embedded) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.started {
		return nil
	}
	executor.Shutdown()
	e.started = false
	return nil
}

func (e *Embedded) Endpoint() Endpoint { return e.endpoint }

func (e *Embedded) Kind() Kind { return KindEmbedded }

func controllerAddr(port int) (string, error) {
	if port <= 0 || port > 65535 {
		port = 0
	}
	// A stopped host releases its listener a moment after Stop returns. Wait
	// briefly so a restart keeps the configured port instead of silently
	// moving the controller to a random one.
	for attempt := 0; attempt < 40; attempt++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			addr := ln.Addr().String()
			_ = ln.Close()
			return addr, nil
		}
		if port == 0 {
			return "", fmt.Errorf("core: pick controller port: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("core: controller port %d is busy: %w", port, err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr, nil
}
func randomSecret() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("core: secret: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// Home is the directory mihomo uses for cache.db, geo data and providers.
func Home(userData string) string { return filepath.Join(userData, "core") }
