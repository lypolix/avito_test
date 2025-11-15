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
	BaseURL     string
	Profile     string
	Duration    time.Duration
	VUs         int
	RPS         int
	ResultsDir  string
	HTTPTimeout time.Duration
}

type Result struct {
	Requests int64   `json:"requests"`
	Errors   int64   `json:"errors"`
	P50      float64 `json:"p50_ms"`
	P95      float64 `json:"p95_ms"`
	P99      float64 `json:"p99_ms"`
	Avg      float64 `json:"avg_ms"`
	Max      float64 `json:"max_ms"`
	Min      float64 `json:"min_ms"`
	StartAt  string  `json:"start_at"`
	Profile  string  `json:"profile"`
	BaseURL  string  `json:"base_url"`
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
	quickSelect(cp)
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

func quickSelect(a []float64) {
	for i := 1; i < len(a); i++ {
		k := a[i]
		j := i - 1
		for j >= 0 && a[j] > k {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = k
	}
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
	fmt.Printf("INFO start profile=%s vus=%d rps=%d duration=%s base=%s\n",
		cfg.Profile, cfg.VUs, cfg.RPS, cfg.Duration, cfg.BaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	cl := newClient(cfg.HTTPTimeout)

	var totalReq, totalErr int64
	timer := NewTimer()

	switch cfg.Profile {
	case "smoke":
		runSmoke(ctx, cl, cfg, &totalReq, &totalErr, timer)
	case "baseline":
		runBaseline(ctx, cl, cfg, &totalReq, &totalErr, timer)
	case "mass":
		runMass(ctx, cl, cfg, &totalReq, &totalErr, timer)
	default:
		fmt.Printf("WARN unknown profile\n")
		runSmoke(ctx, cl, cfg, &totalReq, &totalErr, timer)
	}

	p50, p95, p99, avg, min, max := timer.Summary()
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
	}
	out := filepath.Join(cfg.ResultsDir, fmt.Sprintf("%s-%s.json", cfg.Profile, time.Now().Format("20060102-150405")))
	if err := writeJSON(out, res); err != nil {
		fmt.Printf("ERROR cannot save results: %v\n", err)
	} else {
		fmt.Printf("INFO results saved: %s\n", out)
	}
	fmt.Printf("INFO p95=%.1fms p99=%.1fms err_rate=%.4f reqs=%d\n",
		res.P95, res.P99, rate(totalErr, totalReq), totalReq)
}

func parseFlags() Config {
	var base, profile, duration, out string
	var vus, rps int
	flag.StringVar(&base, "base", "http://localhost:8080", "base URL")
	flag.StringVar(&profile, "profile", "smoke", "profile: smoke|baseline|mass")
	flag.StringVar(&duration, "duration", "1m", "duration (e.g. 1m,5m)")
	flag.StringVar(&out, "out", "loadtest/results", "results dir")
	flag.IntVar(&vus, "vus", 5, "virtual users (goroutines)")
	flag.IntVar(&rps, "rps", 5, "max requests per second overall")
	flag.Parse()

	dur, err := time.ParseDuration(duration)
	if err != nil {
		fmt.Printf("ERROR invalid duration: %v\n", err)
		os.Exit(1)
	}

	return Config{
		BaseURL:     strings.TrimRight(base, "/"),
		Profile:     profile,
		Duration:    dur,
		VUs:         vus,
		RPS:         rps,
		ResultsDir:  out,
		HTTPTimeout: 5 * time.Second,
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

func doJSON(c *http.Client, method, url string, body any, timer *Timer, totalReq, totalErr *int64) ([]byte, int, error) {
	var buf io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	}
	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		atomic.AddInt64(totalErr, 1)
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := c.Do(req)
	elapsed := time.Since(start).Seconds() * 1000
	timer.Add(elapsed)
	atomic.AddInt64(totalReq, 1)

	if err != nil {
		atomic.AddInt64(totalErr, 1)
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		atomic.AddInt64(totalErr, 1)
	}
	return data, resp.StatusCode, nil
}


func runSmoke(ctx context.Context, pool *clientPool, cfg Config, tr *int64, te *int64, timer *Timer) {
	fmt.Println("INFO smoke start")
	seedTeam(pool.c, cfg.BaseURL, "smoke", []member{
		{ID: "u1", Name: "Alice", Active: true},
	})
	runWorkers(ctx, cfg, func(rng *rand.Rand) {
		_, _, _ = doJSON(pool.c, http.MethodGet, cfg.BaseURL+"/health", nil, timer, tr, te)
		_, _, _ = doJSON(pool.c, http.MethodPost, cfg.BaseURL+"/pullRequest/create",
			map[string]any{
				"pull_request_id":   fmt.Sprintf("sm-%d", rng.Int()),
				"pull_request_name": "Test",
				"author_id":         "u1",
			},
			timer, tr, te,
		)
	})
}

type ringBuffer struct {
	ids []string
	mu  sync.Mutex
}

func (r *ringBuffer) pick(rng *rand.Rand) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.ids) == 0 {
		return ""
	}
	return r.ids[rng.Intn(len(r.ids))]
}

func (r *ringBuffer) add(id string, rng *rand.Rand) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.ids) < 64 {
		r.ids = append(r.ids, id)
	} else {
		r.ids[rng.Intn(len(r.ids))] = id
	}
}

func runBaseline(ctx context.Context, pool *clientPool, cfg Config, tr *int64, te *int64, timer *Timer) {
	fmt.Println("INFO baseline start")
	seedTeam(pool.c, cfg.BaseURL, "baseline", []member{
		{ID: "u1", Name: "Alice", Active: true},
		{ID: "u2", Name: "Bob", Active: true},
		{ID: "u3", Name: "Carol", Active: true},
	})

	buf := &ringBuffer{ids: make([]string, 0, 64)}

	runWorkers(ctx, cfg, func(rng *rand.Rand) {
		switch r := rng.Intn(100); {
		case r < 40:
			id := fmt.Sprintf("b-%d-%d", rng.Intn(10_000), rng.Intn(10_000))
			_, _, _ = doJSON(pool.c, http.MethodPost, cfg.BaseURL+"/pullRequest/create",
				map[string]any{
					"pull_request_id":   id,
					"pull_request_name": "Feat",
					"author_id":         "u1",
				}, timer, tr, te)
			buf.add(id, rng)
		case r < 70:
			_, _, _ = doJSON(pool.c, http.MethodGet, cfg.BaseURL+"/users/getReview?user_id=u2", nil, timer, tr, te)
		case r < 90:
			id := buf.pick(rng)
			if id != "" {
				_, _, _ = doJSON(pool.c, http.MethodPost, cfg.BaseURL+"/pullRequest/reassign",
					map[string]any{
						"pull_request_id": id,
						"old_user_id":     "u2",
					}, timer, tr, te)
			}
		default:
			id := buf.pick(rng)
			if id != "" {
				_, _, _ = doJSON(pool.c, http.MethodPost, cfg.BaseURL+"/pullRequest/merge",
					map[string]any{
						"pull_request_id": id,
					}, timer, tr, te)
			}
		}
	})
}

func runMass(ctx context.Context, pool *clientPool, cfg Config, tr *int64, te *int64, timer *Timer) {
	fmt.Println("INFO mass start")
	seedTeam(pool.c, cfg.BaseURL, "ops", []member{
		{ID: "u1", Name: "Alice", Active: true},
		{ID: "u2", Name: "Bob", Active: true},
		{ID: "u3", Name: "Carol", Active: true},
		{ID: "u4", Name: "Dave", Active: true},
	})
	for i := 0; i < 10; i++ {
		_, _, _ = doJSON(pool.c, http.MethodPost, cfg.BaseURL+"/pullRequest/create",
			map[string]any{
				"pull_request_id":   fmt.Sprintf("ops-%d", i),
				"pull_request_name": "Chore",
				"author_id":         "u1",
			}, timer, tr, te)
	}

	runWorkers(ctx, cfg, func(rng *rand.Rand) {
		_, _, _ = doJSON(pool.c, http.MethodGet, cfg.BaseURL+"/users/getReview?user_id=u2", nil, timer, tr, te)
		if rng.Intn(100) < 10 {
			var payload any
			if rng.Intn(2) == 0 {
				payload = map[string]any{"team_name": "ops", "user_ids": []string{"u2", "u3"}}
			} else {
				payload = map[string]any{"team_name": "ops", "user_ids": []string{"u4"}}
			}
			_, _, _ = doJSON(pool.c, http.MethodPost, cfg.BaseURL+"/users/bulkDeactivate", payload, timer, tr, te)
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

func seedTeam(c *http.Client, base, team string, members []member) {
	type m struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsActive bool   `json:"is_active"`
	}
	body := struct {
		TeamName string `json:"team_name"`
		Members  []m    `json:"members"`
	}{
		TeamName: team,
	}
	for _, x := range members {
		body.Members = append(body.Members, m{UserID: x.ID, Username: x.Name, IsActive: x.Active})
	}
	_, code, _ := doJSON(c, http.MethodPost, base+"/team/add", body, NewTimer(), new(int64), new(int64))
	if code != http.StatusCreated && code != http.StatusBadRequest {
		fmt.Printf("WARN failed code=%d\n", code)
	}
}