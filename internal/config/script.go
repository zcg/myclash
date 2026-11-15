package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"gopkg.in/yaml.v3"
)

// ApplyScript runs a profile override script in a small JS runtime:
//
//	function main(config) {
//	  config["dns"] = { enable: true }
//	  return config
//	}
//
// No require/os/net/process APIs are exposed. The runtime is interrupted
// after two seconds so a bad script cannot keep the app from starting.
func ApplyScript(base, script []byte) ([]byte, error) {
	if len(script) == 0 {
		return base, nil
	}
	var doc map[string]any
	if err := yaml.Unmarshal(base, &doc); err != nil {
		return nil, fmt.Errorf("script: parse base: %w", err)
	}
	vm := goja.New()
	_ = vm.Set("console", map[string]any{"log": func(...any) {}})
	_, _ = vm.RunString(string(script))
	main, ok := goja.AssertFunction(vm.Get("main"))
	if !ok {
		return nil, errors.New("script: define function main(config)")
	}
	timer := time.AfterFunc(2*time.Second, func() { vm.Interrupt("script timeout") })
	defer timer.Stop()
	value, err := main(goja.Undefined(), vm.ToValue(doc))
	if err != nil {
		return nil, fmt.Errorf("script: %w", err)
	}
	exported := value.Export()
	out, ok := exported.(map[string]any)
	if !ok {
		return nil, errors.New("script: main(config) must return the config object")
	}
	b, err := yaml.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("script: encode: %w", err)
	}
	return b, nil
}
