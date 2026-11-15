package core

import (
	"fmt"

	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/tunnel"
)

// SetRuleDisabled toggles a rule inside the embedded tunnel. The REST route is
// disabled by embed mode, so the app calls this directly.
func SetRuleDisabled(index int, disabled bool) error {
	rules := tunnel.Rules()
	if index < 0 || index >= len(rules) {
		return fmt.Errorf("core: rule index %d out of range", index)
	}
	w, ok := rules[index].(C.RuleWrapper)
	if !ok {
		return fmt.Errorf("core: rule %d cannot be disabled", index)
	}
	w.SetDisabled(disabled)
	return nil
}

// Validate parses a profile without starting or changing the running core.
func Validate(cfg []byte) error {
	_, err := executor.ParseWithBytes(cfg)
	return err
}
