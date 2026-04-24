package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/prometheus-community/pro-bing"
)

type PingStats struct {
	Avg        time.Duration
	Min        time.Duration
	Max        time.Duration
	Med        time.Duration
	PacketLoss float64
}

type PingResult struct {
	Region Region
	Stats  PingStats
	Error  error
}

const errNoPackets = "no packets received"

func pingHost(host string, count int) (PingStats, error) {
	if runtime.GOOS == "linux" {
		return pingLinux(host, count)
	}

	pinger, err := probing.NewPinger(host)
	if err != nil {
		return PingStats{}, err
	}

	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	} else {
		pinger.SetPrivileged(false)
	}

	pinger.Count = count
	pinger.Timeout = time.Duration(count+2) * time.Second

	var rtts []time.Duration
	pinger.OnRecv = func(pkt *probing.Packet) {
		rtts = append(rtts, pkt.Rtt)
	}

	err = pinger.Run()
	if err != nil {
		return PingStats{}, err
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return PingStats{}, fmt.Errorf(errNoPackets)
	}

	sort.Slice(rtts, func(i, j int) bool {
		return rtts[i] < rtts[j]
	})

	var median time.Duration
	n := len(rtts)
	if n == 0 {
		median = 0
	} else if n%2 == 1 {
		median = rtts[n/2]
	} else {
		median = (rtts[n/2-1] + rtts[n/2]) / 2
	}

	return PingStats{
		Avg:        stats.AvgRtt,
		Min:        stats.MinRtt,
		Max:        stats.MaxRtt,
		Med:        median,
		PacketLoss: stats.PacketLoss,
	}, nil
}

func pingLinux(host string, count int) (PingStats, error) {
	// -c count, -W timeout (seconds)
	cmd := exec.Command("ping", "-c", strconv.Itoa(count), "-W", "5", host)
	out, err := cmd.CombinedOutput()

	outStr := string(out)

	if err != nil && outStr == "" {
		return PingStats{}, err
	}

	// Parse individual ping times for median
	var rtts []time.Duration
	timeRe := regexp.MustCompile(`time=([\d\.]+) ms`)
	matches := timeRe.FindAllStringSubmatch(outStr, -1)
	for _, match := range matches {
		if len(match) == 2 {
			ms, err := strconv.ParseFloat(match[1], 64)
			if err == nil {
				rtts = append(rtts, time.Duration(ms*float64(time.Millisecond)))
			}
		}
	}

	if len(rtts) == 0 {
		return PingStats{}, fmt.Errorf(errNoPackets)
	}

	// Sort for median
	sort.Slice(rtts, func(i, j int) bool {
		return rtts[i] < rtts[j]
	})

	var median time.Duration
	n := len(rtts)
	if n > 0 {
		if n%2 == 1 {
			median = rtts[n/2]
		} else {
			median = (rtts[n/2-1] + rtts[n/2]) / 2
		}
	}

	// Parse packet loss
	lossRe := regexp.MustCompile(`([\d\.]+)% packet loss`)
	lossMatch := lossRe.FindStringSubmatch(outStr)
	var loss float64 = 0
	if len(lossMatch) == 2 {
		loss, _ = strconv.ParseFloat(lossMatch[1], 64)
	}

	// Parse min/avg/max
	statsRe := regexp.MustCompile(`(?:rtt|round-trip) min/avg/max/(?:mdev|stddev) = ([\d\.]+)/([\d\.]+)/([\d\.]+)`)
	statsMatch := statsRe.FindStringSubmatch(outStr)

	var min, avg, max time.Duration
	if len(statsMatch) == 4 {
		minMs, _ := strconv.ParseFloat(statsMatch[1], 64)
		avgMs, _ := strconv.ParseFloat(statsMatch[2], 64)
		maxMs, _ := strconv.ParseFloat(statsMatch[3], 64)
		min = time.Duration(minMs * float64(time.Millisecond))
		avg = time.Duration(avgMs * float64(time.Millisecond))
		max = time.Duration(maxMs * float64(time.Millisecond))
	} else {
		// Fallback if regex fails but we have rtts
		min = rtts[0]
		max = rtts[n-1]
		var total time.Duration
		for _, r := range rtts {
			total += r
		}
		avg = time.Duration(float64(total) / float64(n))
	}

	return PingStats{
		Avg:        avg,
		Min:        min,
		Max:        max,
		Med:        median,
		PacketLoss: loss,
	}, nil
}
