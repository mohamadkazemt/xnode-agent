package health

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func Snapshot() (memoryBytes uint64, load1 float64) {
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		f := strings.Fields(string(b))
		if len(f) > 0 {
			load1, _ = strconv.ParseFloat(f[0], 64)
		}
	}
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	var total, avail uint64
	for s.Scan() {
		p := strings.Fields(s.Text())
		if len(p) < 2 {
			continue
		}
		v, _ := strconv.ParseUint(p[1], 10, 64)
		switch p[0] {
		case "MemTotal:":
			total = v * 1024
		case "MemAvailable:":
			avail = v * 1024
		}
	}
	if total > avail {
		memoryBytes = total - avail
	}
	return
}
