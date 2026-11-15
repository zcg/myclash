package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Merge applies a merge override to a base profile.
//
// Maps merge deeply. Lists replace unless the override uses one of the
// prepend-/append- keys. `delete` removes dot paths.
func Merge(base, override []byte) ([]byte, error) {
	if len(override) == 0 {
		return base, nil
	}
	var b, o map[string]any
	if err := yaml.Unmarshal(base, &b); err != nil {
		return nil, fmt.Errorf("merge: parse base: %w", err)
	}
	if err := yaml.Unmarshal(override, &o); err != nil {
		return nil, fmt.Errorf("merge: parse override: %w", err)
	}
	if b == nil || o == nil {
		return nil, fmt.Errorf("merge: both documents must be YAML mappings")
	}
	for _, key := range []string{"prepend-rules", "append-rules", "prepend-proxies", "append-proxies", "prepend-proxy-groups", "append-proxy-groups", "prepend-listeners", "append-listeners"} {
		v, ok := o[key]
		if !ok {
			continue
		}
		list, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("merge: %s must be a list", key)
		}
		target := targetFor(key)
		dst, _ := b[target].([]any)
		if len(dst) == 0 {
			if existing, ok := b[target].([]any); ok {
				dst = existing
			}
		}
		if len(key) >= 7 && key[:7] == "prepend" {
			b[target] = append(append([]any{}, list...), dst...)
		} else {
			b[target] = append(dst, list...)
		}
		delete(o, key)
	}
	if del, ok := o["delete"].([]any); ok {
		for _, p := range del {
			if path, ok := p.(string); ok {
				deletePath(b, path)
			}
		}
		delete(o, "delete")
	}
	deepMerge(b, o)
	out, err := yaml.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("merge: encode: %w", err)
	}
	return out, nil
}

func targetFor(key string) string {
	switch {
	case contains(key, "rules"):
		return "rules"
	case contains(key, "proxies"):
		return "proxies"
	case contains(key, "proxy-groups"):
		return "proxy-groups"
	case contains(key, "listeners"):
		return "listeners"
	}
	return key
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func deepMerge(dst, src map[string]any) {
	for k, v := range src {
		if vm, ok := v.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				deepMerge(dm, vm)
				continue
			}
		}
		dst[k] = v
	}
}

func deletePath(doc map[string]any, path string) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}
	var cur any = doc
	for _, part := range parts[:len(parts)-1] {
		m, ok := cur.(map[string]any)
		if !ok {
			return
		}
		cur = m[part]
	}
	if m, ok := cur.(map[string]any); ok {
		delete(m, parts[len(parts)-1])
	}
}

func splitPath(path string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			if i > start {
				out = append(out, path[start:i])
			}
			start = i + 1
		}
	}
	return out
}
