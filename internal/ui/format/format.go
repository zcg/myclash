package format

import "fmt"

var units = []string{"B", "KB", "MB", "GB", "TB"}

// Bytes formats a byte count with binary units and one decimal place after KB.
func Bytes(n int64) string {
	if n < 0 {
		n = 0
	}
	f := float64(n)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", n, units[i])
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

func Rate(bytesPerSecond int64) string { return Bytes(bytesPerSecond) + "/s" }

func Percent(part, total int64) string {
	if total <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", float64(part)/float64(total)*100)
}
