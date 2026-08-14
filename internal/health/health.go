package health

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Metrics struct {
	CPUPercent   float64
	MemoryBytes  uint64
	Load1        float64
	NetworkRX    uint64
	NetworkTX    uint64
	NetworkRXBPS float64
	NetworkTXBPS float64
}

type Sampler struct {
	mu       sync.Mutex
	iface    string
	lastCPU  cpuTimes
	lastNet  netTimes
	lastTime time.Time
	primed   bool
}

type cpuTimes struct {
	total uint64
	idle  uint64
}

type netTimes struct {
	rx uint64
	tx uint64
}

func NewSampler(iface string) *Sampler { return &Sampler{iface: iface} }

func (s *Sampler) Snapshot(now time.Time) Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()

	var m Metrics
	m.MemoryBytes, m.Load1 = memoryAndLoad()
	cpu, cpuErr := readCPU()
	netv, netErr := readNetwork(s.iface)

	if s.primed {
		if cpuErr == nil && cpu.total > s.lastCPU.total {
			totalDelta := cpu.total - s.lastCPU.total
			idleDelta := cpu.idle - s.lastCPU.idle
			if idleDelta <= totalDelta {
				m.CPUPercent = 100 * float64(totalDelta-idleDelta) / float64(totalDelta)
			}
		}
		elapsed := now.Sub(s.lastTime).Seconds()
		if elapsed > 0 && netErr == nil {
			if netv.rx >= s.lastNet.rx {
				m.NetworkRXBPS = float64(netv.rx-s.lastNet.rx) * 8 / elapsed
			}
			if netv.tx >= s.lastNet.tx {
				m.NetworkTXBPS = float64(netv.tx-s.lastNet.tx) * 8 / elapsed
			}
		}
	}
	if cpuErr == nil {
		s.lastCPU = cpu
	}
	if netErr == nil {
		s.lastNet = netv
		m.NetworkRX = netv.rx
		m.NetworkTX = netv.tx
	}
	s.lastTime = now
	s.primed = true
	return m
}

func Snapshot() (memoryBytes uint64, load1 float64) { return memoryAndLoad() }

func memoryAndLoad() (memoryBytes uint64, load1 float64) {
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
	sc := bufio.NewScanner(f)
	var total, avail uint64
	for sc.Scan() {
		p := strings.Fields(sc.Text())
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

func readCPU() (cpuTimes, error) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	line := strings.SplitN(string(b), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, errors.New("invalid /proc/stat")
	}
	vals := make([]uint64, 0, len(fields)-1)
	for _, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuTimes{}, err
		}
		vals = append(vals, v)
	}
	var total uint64
	for _, v := range vals {
		total += v
	}
	idle := vals[3]
	if len(vals) > 4 {
		idle += vals[4] // iowait
	}
	return cpuTimes{total: total, idle: idle}, nil
}

func readNetwork(iface string) (netTimes, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return netTimes{}, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	var total netTimes
	found := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		i := strings.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		name := strings.TrimSpace(line[:i])
		if name == "lo" {
			continue
		}
		if iface != "" && name != iface {
			continue
		}
		fields := strings.Fields(line[i+1:])
		if len(fields) < 16 {
			continue
		}
		rx, e1 := strconv.ParseUint(fields[0], 10, 64)
		tx, e2 := strconv.ParseUint(fields[8], 10, 64)
		if e1 != nil || e2 != nil {
			continue
		}
		total.rx += rx
		total.tx += tx
		found = true
	}
	if err := sc.Err(); err != nil {
		return netTimes{}, err
	}
	if !found {
		return netTimes{}, errors.New("network interface not found")
	}
	return total, nil
}
