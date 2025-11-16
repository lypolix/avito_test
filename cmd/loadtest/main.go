package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	BaseURL        string
	Profile        string
	Duration       time.Duration
	Warmup         time.Duration
	VUs            int
	RPS            int
	ResultsDir     string
	HTTPTimeout    time.Duration
	Count4xxAsErrs bool
}

type Result struct {
	Requests int64              `json:"requests"`
	Errors   int64              `json:"errors"`
	P50      float64            `json:"p50_ms"`
	P95      float64            `json:"p95_ms"`
	P99      float64            `json:"p99_ms"`
	Avg      float64            `json:"avg_ms"`
	Max      float64            `json:"max_ms"`
	Min      float64            `json:"min_ms"`
	StartAt  string             `json:"start_at"`
	Profile  string             `json:"profile"`
	BaseURL  string             `json:"base_url"`
	ByPath   map[string]ByClass `json:"by_path"`
}

type ByClass struct {
	Code2xx int `json:"2xx"`
	Code4xx int `json:"4xx"`
	Code5xx int `json:"5xx"`
}

type Timer struct {
	mu   sync.Mutex
	vals []float64
	min  float64
	max  float64
	sum  float64
}

func NewTimer() *Timer { return &Timer{min: 1e18} }

func (t *Timer) Add(ms float64) {
	t.mu.Lock()
	t.vals = append(t.vals, ms)
	if ms < t.min {
		t.min = ms
	}
	if ms > t.max {
		t.max = ms
	}
	t.sum += ms
	t.mu.Unlock()
}

func (t *Timer) Summary() (p50, p95, p99, avg, min, max float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := len(t.vals)
	if n == 0 {
		return 0, 0, 0, 0, 0, 0
	}
	cp := make([]float64, n)
	copy(cp, t.vals)
	for i := 1; i < len(cp); i++ {
		k := cp[i]
		j := i - 1
		for j >= 0 && cp[j] > k {
			cp[j+1] = cp[j]
			j--
		}
		cp[j+1] = k
	}
	get := func(p float64) float64 {
		idx := int(float64(n-1) * p)
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		return cp[idx]
	}
	p50 = get(0.50)
	p95 = get(0.95)
	p99 = get(0.99)
	avg = t.sum / float64(n)
	return p50, p95, p99, avg, t.min, t.max
}

type clientPool struct {
	c *http.Client
}

func newClient(timeout time.Duration) *clientPool {
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 3 * time.Second,
		DisableCompression:  false,
	}
	return &clientPool{
		c: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func main() {
	cfg := parseFlags()
	if err := os.MkdirAll(cfg.ResultsDir, 0o755); err != nil {
		fmt.Printf("ERROR cannot create results dir: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("INFO start profile=%s vus=%d rps=%d warmup=%s duration=%s base=%s\n",
		cfg.Profile, cfg.VUs, cfg.RPS, cfg.Warmup, cfg.Duration, cfg.BaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Warmup+cfg.Duration)
	defer cancel()

	cl := newClient(cfg.HTTPTimeout)

	var totalReq, totalErr int64
	timer := NewTimer()

	stats := struct {
		mu   sync.Mutex
		data map[string]ByClass
	}{data: make(map[string]ByClass)}

	var collect atomic.Bool

	go func() {
		if cfg.Warmup > 0 {
			time.Sleep(cfg.Warmup)
		}
		collect.Store(true)
	}()

	call := func(method, path string, body any) ([]byte, int, error) {
		start := time.Now()
		b, code, err := doJSON(cl.c, method, cfg.BaseURL+path, body)
		elapsed := time.Since(start).Seconds() * 1000
		if collect.Load() {
			timer.Add(elapsed)
			atomic.AddInt64(&totalReq, 1)
			class := classify(code, err)
			stats.mu.Lock()
			s := stats.data[path]
			switch class {
			case 2:
				s.Code2xx++
			case 4:
				s.Code4xx++
			case 5:
				s.Code5xx++
			}
			stats.data[path] = s
			stats.mu.Unlock()
			if cfg.Count4xxAsErrs {
				if err != nil || code >= 400 {
					atomic.AddInt64(&totalErr, 1)
				}
			} else {
				if err != nil || code >= 500 {
					atomic.AddInt64(&totalErr, 1)
				}
			}
		}
		return b, code, err
	}

	switch cfg.Profile {
	case "smoke":
		runSmoke(ctx, call, cfg)
	case "baseline":
		runBaseline(ctx, call, cfg)
	case "mass":
		runMass(ctx, call, cfg)
	default:
		fmt.Printf("WARN unknown profile\n")
		runSmoke(ctx, call, cfg)
	}

	p50, p95, p99, avg, min, max := timer.Summary()
	byPath := make(map[string]ByClass)
	stats.mu.Lock()
	for k, v := range stats.data {
		byPath[k] = v
	}
	stats.mu.Unlock()

	res := Result{
		Requests: totalReq,
		Errors:   totalErr,
		P50:      p50,
		P95:      p95,
		P99:      p99,
		Avg:      avg,
		Min:      min,
		Max:      max,
		StartAt:  time.Now().Format(time.RFC3339),
		Profile:  cfg.Profile,
		BaseURL:  cfg.BaseURL,
		ByPath:   byPath,
	}
	out := filepath.Join(cfg.ResultsDir, fmt.Sprintf("%s-%s.json", cfg.Profile, time.Now().Format("20060102-150405")))
	if err := writeJSON(out, res); err != nil {
		fmt.Printf("ERROR cannot save results: %v\n", err)
	} else {
		fmt.Printf("INFO results saved: %s\n", out)
	}
	errRate := rate(totalErr, totalReq)
	fmt.Printf("INFO p95=%.1fms p99=%.1fms err_rate=%.4f reqs=%d\n", res.P95, res.P99, errRate, res.Requests)
	printByPath(byPath)
}

func parseFlags() Config {
	var base, profile, duration, warmup, out string
	var vus, rps int
	var count4xx bool
	flag.StringVar(&base, "base", "http://localhost:18080", "base URL")
	flag.StringVar(&profile, "profile", "baseline", "profile: smoke|baseline|mass")
	flag.StringVar(&duration, "duration", "3m", "main duration (e.g. 1m,5m)")
	flag.StringVar(&warmup, "warmup", "20s", "warmup period (excluded from metrics)")
	flag.StringVar(&out, "out", "loadtest/results", "results dir")
	flag.IntVar(&vus, "vus", 10, "virtual users (goroutines)")
	flag.IntVar(&rps, "rps", 5, "max requests per second overall")
	flag.BoolVar(&count4xx, "count4xx", false, "count 4xx as errors in SLI")
	flag.Parse()

	dur, err := time.ParseDuration(duration)
	if err != nil {
		fmt.Printf("ERROR invalid duration: %v\n", err)
		os.Exit(1)
	}
	wu, err := time.ParseDuration(warmup)
	if err != nil {
		fmt.Printf("ERROR invalid warmup: %v\n", err)
		os.Exit(1)
	}

	return Config{
		BaseURL:        strings.TrimRight(base, "/"),
		Profile:        profile,
		Duration:       dur,
		Warmup:         wu,
		VUs:            vus,
		RPS:            rps,
		ResultsDir:     out,
		HTTPTimeout:    5 * time.Second,
		Count4xxAsErrs: count4xx,
	}
}

func rate(errs, reqs int64) float64 {
	if reqs == 0 {
		return 0
	}
	return float64(errs) / float64(reqs)
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func throttle(rps int) <-chan time.Time {
	if rps <= 0 {
		ch := make(chan time.Time)
		close(ch)
		return ch
	}
	return time.NewTicker(time.Second / time.Duration(rps)).C
}

func doJSON(c *http.Client, method, url string, body any) ([]byte, int, error) {
	var buf io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	}
	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

func classify(code int, err error) int {
	if err != nil {
		return 5
	}
	switch {
	case code >= 500:
		return 5
	case code >= 400:
		return 4
	default:
		return 2
	}
}

func printByPath(m map[string]ByClass) {
	if len(m) == 0 {
		return
	}
	fmt.Println("INFO per-path status summary:")
	for p, s := range m {
		fmt.Printf("  %s -> 2xx=%d 4xx=%d 5xx=%d\n", p, s.Code2xx, s.Code4xx, s.Code5xx)
	}
}

func runSmoke(ctx context.Context, call func(method, path string, body any) ([]byte, int, error), cfg Config) {
	fmt.Println("INFO smoke start")
	seedTeam(call, "smoke", []member{
		{ID: "u1", Name: "Alice", Active: true},
	})
	runWorkers(ctx, cfg, func(rng *rand.Rand) {
		_, _, _ = call(http.MethodGet, "/health", nil)
		_, _, _ = call(http.MethodPost, "/pullRequest/create", map[string]any{
			"pull_request_id":   fmt.Sprintf("sm-%d", rng.Int()),
			"pull_request_name": "Test",
			"author_id":         "u1",
		})
	})
}

type ringBuffer struct {
	ids map[string]struct{}
	mu  sync.Mutex
}

func newRing() *ringBuffer { return &ringBuffer{ids: make(map[string]struct{}, 64)} }
func (r *ringBuffer) pick(rng *rand.Rand) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(r.ids)
	if n == 0 {
		return ""
	}
	idx := rng.Intn(n)
	i := 0
	for id := range r.ids {
		if i == idx {
			return id
		}
		i++
	}
	return ""
}

func (r *ringBuffer) add(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.ids) >= 64 {
		for k := range r.ids {
			delete(r.ids, k)
			break
		}
	}
	r.ids[id] = struct{}{}
}

func (r *ringBuffer) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.ids, id)
}

func runBaseline(ctx context.Context, call func(method, path string, body any) ([]byte, int, error), cfg Config) {
	fmt.Println("INFO baseline start")
	seedTeam(call, "baseline", []member{
		{ID: "u1", Name: "Alice", Active: true},
		{ID: "u2", Name: "Bob", Active: true},
		{ID: "u3", Name: "Carol", Active: true},
	})
	buf := newRing()

	runWorkers(ctx, cfg, func(rng *rand.Rand) {
		switch r := rng.Intn(100); {
		case r < 40:
			id := fmt.Sprintf("b-%d-%d", rng.Intn(10_000), rng.Intn(10_000))
			_, _, _ = call(http.MethodPost, "/pullRequest/create", map[string]any{
				"pull_request_id":   id,
				"pull_request_name": "Feat",
				"author_id":         "u1",
			})
			buf.add(id)
		case r < 70:
			_, _, _ = call(http.MethodGet, "/users/getReview?user_id=u2", nil)
		case r < 90:
			id := buf.pick(rng)
			if id != "" {
				_, _, _ = call(http.MethodPost, "/pullRequest/reassign", map[string]any{
					"pull_request_id": id,
					"old_user_id":     "u2",
				})
			}
		default:
			id := buf.pick(rng)
			if id != "" {
				_, code, _ := call(http.MethodPost, "/pullRequest/merge", map[string]any{
					"pull_request_id": id,
				})
				if code >= 200 && code < 300 {
					buf.remove(id)
				}
			}
		}
	})
}

func runMass(ctx context.Context, call func(method, path string, body any) ([]byte, int, error), cfg Config) {
	fmt.Println("INFO mass start")
	seedTeam(call, "ops", []member{
		{ID: "u1", Name: "Alice", Active: true},
		{ID: "u2", Name: "Bob", Active: true},
		{ID: "u3", Name: "Carol", Active: true},
		{ID: "u4", Name: "Dave", Active: true},
	})
	for i := 0; i < 10; i++ {
		_, _, _ = call(http.MethodPost, "/pullRequest/create", map[string]any{
			"pull_request_id":   fmt.Sprintf("ops-%d", i),
			"pull_request_name": "Chore",
			"author_id":         "u1",
		})
	}
	runWorkers(ctx, cfg, func(rng *rand.Rand) {
		_, _, _ = call(http.MethodGet, "/users/getReview?user_id=u2", nil)
		if rng.Intn(100) < 10 {
			var payload any
			if rng.Intn(2) == 0 {
				payload = map[string]any{"team_name": "ops", "user_ids": []string{"u2", "u3"}}
			} else {
				payload = map[string]any{"team_name": "ops", "user_ids": []string{"u4"}}
			}
			_, _, _ = call(http.MethodPost, "/users/bulkDeactivate", payload)
		}
	})
}

func runWorkers(ctx context.Context, cfg Config, work func(rng *rand.Rand)) {
	var wg sync.WaitGroup
	th := throttle(cfg.RPS)
	for i := 0; i < cfg.VUs; i++ {
		wg.Add(1)
		seed := time.Now().UnixNano() + int64(i+1)
		rng := rand.New(rand.NewSource(seed))
		go func(id int, rng *rand.Rand) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-th:
					work(rng)
				}
			}
		}(i+1, rng)
	}
	wg.Wait()
}

type member struct {
	ID     string
	Name   string
	Active bool
}

func seedTeam(call func(method, path string, body any) ([]byte, int, error), team string, members []member) {
	type m struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsActive bool   `json:"is_active"`
	}
	body := struct {
		TeamName string `json:"team_name"`
		Members  []m    `json:"members"`
	}{TeamName: team}
	for _, x := range members {
		body.Members = append(body.Members, m{UserID: x.ID, Username: x.Name, IsActive: x.Active})
	}
	_, code, _ := call(http.MethodPost, "/team/add", body)
	if code != http.StatusCreated && code != http.StatusBadRequest {
		fmt.Printf("WARN seed team failed code=%d\n", code)
	}
}
