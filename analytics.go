package main

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DayStat is one day's nginx access-log aggregate for a site.
type DayStat struct {
	Date   string // "2026-05-22"
	Label  string // "22/05" (short label for charts)
	Hits   int
	Bytes  int64
	Unique int
}

// ChartBar is a precomputed bar in the Analytics SVG so the template doesn't
// have to do any arithmetic.
type ChartBar struct {
	Label  string
	Hits   int
	X      int // bar centre x
	BarX   int // bar left x (X - half width)
	Y      int // bar top y
	H      int // bar height
	LabelY int // y for the hits label above the bar
}

// SiteAnalytics is what the Analytics tab renders.
type SiteAnalytics struct {
	Days       []DayStat
	Bars       []ChartBar
	TotalHits  int
	TotalBytes int64
	UniqueHits int
	MaxHits    int
	MaxBytes   int64
	Window     string // "last 7 days", etc.
	HumanBytes string
	HasLog     bool
}

// nginx combined-format line:
//   1.2.3.4 - user [27/May/2026:10:46:21 +0000] "GET /path HTTP/2.0" 200 1234 "ref" "ua"
var nginxLineRe = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "[^"]*" (\d{3}) (\d+|-)`)

const nginxTimeLayout = "02/Jan/2006:15:04:05 -0700"

// siteAnalytics parses the last `days` days of /var/log/nginx/<domain>.access.log
// plus any rotated gzipped copies. Returns per-day counts even when no log
// file exists yet (zeroed entries so the chart still draws a frame).
func (a *App) siteAnalytics(domain string, days int) SiteAnalytics {
	if days <= 0 {
		days = 7
	}
	now := time.Now().UTC()
	earliest := now.AddDate(0, 0, -(days - 1))

	buckets := map[string]*DayStat{}
	uniqueByDay := map[string]map[string]bool{}
	dateKeys := make([]string, 0, days)
	for i := 0; i < days; i++ {
		t := earliest.AddDate(0, 0, i)
		key := t.Format("2006-01-02")
		dateKeys = append(dateKeys, key)
		buckets[key] = &DayStat{Date: key, Label: t.Format("02/01")}
		uniqueByDay[key] = map[string]bool{}
	}

	result := SiteAnalytics{Window: "last " + strconv.Itoa(days) + " days"}

	base := "/var/log/nginx/" + domain + ".access.log"
	files := []string{base}
	if entries, err := os.ReadDir("/var/log/nginx/"); err == nil {
		prefix := domain + ".access.log."
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), prefix) {
				files = append(files, "/var/log/nginx/"+e.Name())
			}
		}
	}
	sort.Strings(files)

	anyRead := false
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		anyRead = true
		var r io.Reader = f
		if strings.HasSuffix(path, ".gz") {
			gz, err := gzip.NewReader(f)
			if err != nil {
				_ = f.Close()
				continue
			}
			r = gz
		}
		s := bufio.NewScanner(r)
		s.Buffer(make([]byte, 64*1024), 1024*1024)
		for s.Scan() {
			m := nginxLineRe.FindStringSubmatch(s.Text())
			if m == nil {
				continue
			}
			ip, raw, _, bytesStr := m[1], m[2], m[3], m[4]
			t, err := time.Parse(nginxTimeLayout, raw)
			if err != nil {
				continue
			}
			key := t.UTC().Format("2006-01-02")
			b, ok := buckets[key]
			if !ok {
				continue
			}
			b.Hits++
			result.TotalHits++
			if n, err := strconv.ParseInt(bytesStr, 10, 64); err == nil {
				b.Bytes += n
				result.TotalBytes += n
				if n > result.MaxBytes {
					result.MaxBytes = n
				}
			}
			if !uniqueByDay[key][ip] {
				uniqueByDay[key][ip] = true
				b.Unique++
			}
		}
		_ = f.Close()
	}

	allUnique := map[string]bool{}
	for _, key := range dateKeys {
		for ip := range uniqueByDay[key] {
			allUnique[ip] = true
		}
	}
	result.UniqueHits = len(allUnique)
	result.HasLog = anyRead

	for _, key := range dateKeys {
		b := *buckets[key]
		if b.Hits > result.MaxHits {
			result.MaxHits = b.Hits
		}
		result.Days = append(result.Days, b)
	}
	result.HumanBytes = humanSize(result.TotalBytes)

	// Precompute the SVG bar geometry so the template doesn't have to do
	// any arithmetic across int/int64 types. viewBox is 0 0 700 260; the
	// drawable area is x: 40..690, y: 40..220 (height 180).
	const (
		chartLeft   = 40
		chartRight  = 690
		chartTop    = 40
		chartBottom = 220
		chartHeight = chartBottom - chartTop
		barWidth    = 32
	)
	usable := chartRight - chartLeft
	max := result.MaxHits
	if max < 1 {
		max = 1
	}
	if n := len(result.Days); n > 0 {
		step := usable / n
		for i, d := range result.Days {
			x := chartLeft + step/2 + i*step
			h := d.Hits * chartHeight / max
			y := chartBottom - h
			result.Bars = append(result.Bars, ChartBar{
				Label:  d.Label,
				Hits:   d.Hits,
				X:      x,
				BarX:   x - barWidth/2,
				Y:      y,
				H:      h,
				LabelY: y - 6,
			})
		}
	}
	return result
}
