package app

import (
	"testing"

	"myclash/internal/platform"
)

func TestPrepareRestartStopsCoreAndRestoresProxy(t *testing.T) {
	a := newTestApp(t)
	a.proxyOwned = true
	a.systemProxy = true
	sp := &fakeSysProxy{}
	a.sysproxy = sp

	a.prepareRestart()

	if a.host != nil || a.client != nil {
		t.Fatalf("core should be cleared before restart: host=%v client=%v", a.host, a.client)
	}
	if a.systemProxy {
		t.Fatal("system proxy should be restored before restart")
	}
	if got := a.state.View().Started; got {
		t.Fatal("state should report the core stopped")
	}
	if a.proxyOwned {
		t.Fatal("proxy ownership should be cleared")
	}
	if sp.restored != (platform.ProxySnapshot{}) {
		t.Fatalf("proxy snapshot not restored: %+v", sp.restored)
	}
}

func TestPrepareRestartIsIdempotent(t *testing.T) {
	a := newTestApp(t)
	a.prepareRestart()
	a.prepareRestart()
}

func TestFinishShutdownRestoresSystemProxyOnlyWhenOwned(t *testing.T) {
	t.Run("owned", func(t *testing.T) {
		a := newTestApp(t)
		a.proxyOwned = true
		a.systemProxy = true
		sp := &fakeSysProxy{}
		a.sysproxy = sp
		a.finishShutdown()
		if sp.restored != (platform.ProxySnapshot{}) {
			t.Fatalf("snapshot not restored: %+v", sp.restored)
		}
	})
	t.Run("not owned", func(t *testing.T) {
		a := newTestApp(t)
		a.systemProxy = true
		sp := &fakeSysProxy{}
		a.sysproxy = sp
		a.finishShutdown()
		if sp.restored != (platform.ProxySnapshot{}) {
			t.Fatalf("foreign proxy must not be restored: %+v", sp.restored)
		}
	})
}

type fakeSysProxy struct {
	restored platform.ProxySnapshot
}

func (f *fakeSysProxy) Enable(string, int, []string) error { return nil }
func (f *fakeSysProxy) Disable() error                     { return nil }
func (f *fakeSysProxy) Enabled() (bool, error)             { return false, nil }
func (f *fakeSysProxy) Snapshot() (platform.ProxySnapshot, error) {
	return platform.ProxySnapshot{}, nil
}
func (f *fakeSysProxy) Restore(s platform.ProxySnapshot) error {
	f.restored = s
	return nil
}
