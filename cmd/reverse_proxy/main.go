package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Alive = alive
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.Alive
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

func NewServerPool() *ServerPool {
	return &ServerPool{
		backends: make([]*Backend, 0),
		current:  0,
	}
}

func (s *ServerPool) AddBackend(b *Backend) {
	s.backends = append(s.backends, b)
}

func (s *ServerPool) GetNextHealthyBackend() *Backend {
	numBackends := uint64(len(s.backends))
	if numBackends == 0 {
		return nil
	}

	for i := uint64(0); i < numBackends; i++ {
		idx := atomic.AddUint64(&s.current, 1) - 1
		backendIdx := idx % numBackends
		backend := s.backends[backendIdx]
		if backend.IsAlive() {
			return backend
		}
	}
	return nil
}

func (s *ServerPool) HealthCheck(healthCheckInterval time.Duration) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		log.Println("Proxy: Running health checks...")
		for _, b := range s.backends {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

			healthURL := b.URL.ResolveReference(&url.URL{Path: "/health"})
			req, err := http.NewRequestWithContext(ctx, "GET", healthURL.String(), nil)
			if err != nil {
				log.Printf("Proxy: Error creating health check request for %s: %v\n", b.URL, err)
				b.SetAlive(false)
				cancel()
				continue
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("Proxy: Backend %s failed health check (connection error): %v\n", b.URL, err)
				b.SetAlive(false)
			} else {
				if resp.StatusCode == http.StatusOK {
					if !b.IsAlive() {
						log.Printf("Proxy: Backend %s is now ALIVE.\n", b.URL)
					}
					b.SetAlive(true)
				} else {
					if b.IsAlive() {
						log.Printf("Proxy: Backend %s failed health check (status: %d).\n", b.URL, resp.StatusCode)
					}
					b.SetAlive(false)
				}
				resp.Body.Close()
			}
			cancel()
		}
	}
}

func serveProxy(w http.ResponseWriter, r *http.Request, pool *ServerPool) {
	backend := pool.GetNextHealthyBackend()
	if backend == nil {
		log.Println("Proxy: No healthy backend available.")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	log.Printf("Proxy: Forwarding request %s %s to %s\n", r.Method, r.URL.Path, backend.URL)
	backend.ReverseProxy.ServeHTTP(w, r)
}

func main() {
	backendURLsStr := os.Getenv("TARGET_BACKEND_URLS")
	if backendURLsStr == "" {
		backendURLsStr = "http://localhost:8080"
	}
	urls := strings.Split(backendURLsStr, ",")

	proxyListenAddr := os.Getenv("PROXY_LISTEN_ADDR")
	if proxyListenAddr == "" {
		proxyListenAddr = ":9090"
	}

	healthCheckIntervalStr := os.Getenv("HEALTH_CHECK_INTERVAL")
	healthCheckInterval, err := time.ParseDuration(healthCheckIntervalStr)
	if err != nil || healthCheckInterval <= 0 {
		healthCheckInterval = 10 * time.Second
	}

	pool := NewServerPool()

	for _, urlStr := range urls {
		trimmedURL := strings.TrimSpace(urlStr)
		if trimmedURL == "" {
			continue
		}
		backendURL, err := url.Parse(trimmedURL)
		if err != nil {
			log.Fatalf("Proxy: Error parsing backend URL '%s': %v", trimmedURL, err)
		}

		rp := httputil.NewSingleHostReverseProxy(backendURL)

		originalDirector := rp.Director
		rp.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = backendURL.Host // Ensure correct host header for backend
		}

		rp.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			log.Printf("Proxy: Error connecting to backend %s: %v", backendURL, err)
			for _, b := range pool.backends {
				if b.URL.String() == backendURL.String() {
					b.SetAlive(false)
					break
				}
			}
			http.Error(rw, "Error communicating with target server.", http.StatusBadGateway)
		}

		backend := &Backend{
			URL:          backendURL,
			Alive:        false,
			ReverseProxy: rp,
		}
		pool.AddBackend(backend)
		log.Printf("Proxy: Backend %s added to pool.\n", backendURL)
	}

	if len(pool.backends) == 0 {
		log.Fatal("Proxy: No backends configured. Exiting.")
	}

	go pool.HealthCheck(healthCheckInterval)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveProxy(w, r, pool)
	})

	log.Printf("🚀 Starting reverse proxy on %s, health check interval: %s\n", proxyListenAddr, healthCheckInterval)
	if err := http.ListenAndServe(proxyListenAddr, nil); err != nil {
		log.Fatalf("Proxy: Error starting proxy server: %v", err)
	}
}
