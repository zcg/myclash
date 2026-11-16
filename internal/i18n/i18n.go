// Package i18n keeps the app's language independent of individual pages.
// The source strings are Chinese; English is selected at runtime.
package i18n

import "sync/atomic"

var lang atomic.Value // string

func init() { lang.Store("zh-CN") }

func Set(code string) {
	switch code {
	case "en":
		lang.Store("en")
	default:
		lang.Store("zh-CN")
	}
}

func English() bool { return lang.Load() == "en" }

// T returns the English string when the language is English, otherwise the
// source Chinese string.
func T(zh, en string) string {
	if English() {
		return en
	}
	return zh
}
