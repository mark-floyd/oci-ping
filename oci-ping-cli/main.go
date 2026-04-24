package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/prometheus-community/pro-bing"
	"github.com/schollz/progressbar/v3"
)

type Region struct {
	RegionName      string `json:"regionName"`
	RegionLocation  string `json:"regionLocation"`
	PingURL         string `json:"pingUrl"`
	RegionContinent string `json:"regionContinent"`
}

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

func main() {
	verbose := flag.Bool("v", false, "enable verbose output")
	pingCount := flag.Int("n", 10, "number of pings to each region")
	regionsList := flag.String("regions-list", "https://ghfast.top/raw.githubusercontent.com/mark-floyd/oci-ping/refs/heads/main/regions.json", "path or URL to the regions JSON file")
	saveCSV := flag.Bool("save", false, "save results to a CSV file")
	flag.Parse()

	fmt.Printf("OCI Ping CLI (%s/%s)\n", runtime.GOOS, runtime.GOARCH)

	regions, err := loadRegions(*regionsList)
	if err != nil {
		log.Fatalf("Failed to load regions from %s: %v", *regionsList, err)
	}

	fmt.Printf("Successfully loaded %d regions. Starting ICMP pings (%d per region)...\n", len(regions), *pingCount)
	fmt.Println("")

	bar := progressbar.NewOptions(len(regions),
		progressbar.OptionSetDescription("Pinging"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Println("\n")
		}),
	)

	resultsChan := make(chan PingResult, len(regions))
	var wg sync.WaitGroup

	for _, r := range regions {
		wg.Add(1)
		go func(reg Region) {
			defer wg.Done()
			stats, err := pingHost(reg.PingURL, *pingCount)
			resultsChan <- PingResult{Region: reg, Stats: stats, Error: err}
			bar.Add(1)
		}(r)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var results []PingResult
	for res := range resultsChan {
		results = append(results, res)
	}

	// Sort by average latency (ascending), with errors at the end
	sort.Slice(results, func(i, j int) bool {
		if results[i].Error != nil && results[j].Error == nil {
			return false
		}
		if results[i].Error == nil && results[j].Error != nil {
			return true
		}
		if results[i].Error != nil && results[j].Error != nil {
			return results[i].Region.RegionName < results[j].Region.RegionName
		}
		return results[i].Stats.Avg < results[j].Stats.Avg
	})

	// Prepare table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Region Name", "Continent", "Average", "Median", "Minimum", "Maximum", "Packet Loss"})
	table.SetBorder(true)
	table.SetAutoWrapText(false)

	var csvWriter *csv.Writer
	var csvFile *os.File
	var csvFileName string

	if *saveCSV {
		timestamp := time.Now().Format("2006-01-02T15-04-05")
		csvFileName = fmt.Sprintf("%s-results.csv", timestamp)
		csvFile, err = os.Create(csvFileName)
		if err != nil {
			log.Printf("Warning: Could not create %s: %v", csvFileName, err)
		} else {
			defer csvFile.Close()
			csvWriter = csv.NewWriter(csvFile)
			defer csvWriter.Flush()
			csvWriter.Write([]string{"Region Name", "Continent", "Average (ms)", "Median (ms)", "Minimum (ms)", "Maximum (ms)", "Packet Loss (%)"})
		}
	}

	lossCount := 0
	otherErrorCount := 0
	for _, res := range results {
		if res.Error != nil {
			if res.Error.Error() == errNoPackets {
				lossCount++
			} else {
				otherErrorCount++
			}

			// Always show in table, and CSV if enabled
			table.Append([]string{
				res.Region.RegionName,
				res.Region.RegionContinent,
				"Error", "Error", "Error", "Error", "100.00%",
			})

			if csvWriter != nil {
				csvWriter.Write([]string{res.Region.RegionName, res.Region.RegionContinent, "Error", "Error", "Error", "Error", "100.00"})
			}
			continue
		}

		avgVal := formatDuration(res.Stats.Avg)
		minVal := formatDuration(res.Stats.Min)
		maxVal := formatDuration(res.Stats.Max)
		medVal := formatDuration(res.Stats.Med)
		lossVal := fmt.Sprintf("%.2f%%", res.Stats.PacketLoss)

		// Console Table
		tableRow := []string{
			res.Region.RegionName,
			res.Region.RegionContinent,
			avgVal + " ms",
			medVal + " ms",
			minVal + " ms",
			maxVal + " ms",
			lossVal,
		}

		colors := []tablewriter.Colors{
			{}, {}, 
			getColor(res.Stats.Avg),
			getColor(res.Stats.Med),
			getColor(res.Stats.Min),
			getColor(res.Stats.Max),
			{}, // Packet Loss (no color)
		}
		table.Rich(tableRow, colors)

		// CSV File (numeric only)
		if csvWriter != nil {
			csvWriter.Write([]string{
				res.Region.RegionName,
				res.Region.RegionContinent,
				avgVal,
				medVal,
				minVal,
				maxVal,
				fmt.Sprintf("%.2f", res.Stats.PacketLoss),
			})
		}
	}

	table.Render()

	if lossCount > 0 && !*verbose {
		regionWord := "region"
		if lossCount > 1 {
			regionWord = "regions"
		}
		fmt.Printf("\nNote: %d %s failed to respond. Use -v for details.\n", lossCount, regionWord)
	}

	if otherErrorCount > 0 && !*verbose {
		errWord := "error"
		if otherErrorCount > 1 {
			errWord = "errors"
		}
		fmt.Printf("Note: %d %s encountered during pings. Use -v for details.\n", otherErrorCount, errWord)
	}

	if csvFile != nil {
		fmt.Printf("\nResults saved to %s\n", csvFileName)
	}
}

func getColor(d time.Duration) tablewriter.Colors {
	if runtime.GOOS == "windows" {
		return tablewriter.Colors{}
	}
	ms := float64(d.Milliseconds())
	if ms < 100 {
		return tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor}
	} else if ms < 200 {
		return tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiYellowColor}
	} else if ms < 300 {
		return tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor}
	} else {
		return tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor}
	}
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2f", float64(d.Microseconds())/1000.0)
}

func loadRegions(location string) ([]Region, error) {
	var body []byte
	var err error

	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		fmt.Printf("Reading regions.json from %s\n", location)
		resp, err := http.Get(location)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		body, err = os.ReadFile(location)
		if err != nil {
			return nil, err
		}
	}

	var regions []Region
	err = json.Unmarshal(body, &regions)
	if err != nil {
		return nil, err
	}

	return regions, nil
}

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
