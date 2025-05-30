package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type proxyServer struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

func newProxyServer(targetURL string) (*proxyServer, error) {
	parsedTargetURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing target URL '%s': %w", targetURL, err)
	}

	p := httputil.NewSingleHostReverseProxy(parsedTargetURL)

	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)

		log.Printf("Proxy: Forwarding request %s %s to %s%s\n",
			req.Method, req.Host, parsedTargetURL.Host, req.URL.Path)
	}

	return &proxyServer{
		target: parsedTargetURL,
		proxy:  p,
	}, nil
}

func (ps *proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ps.proxy.ServeHTTP(w, r)
}

func main() {
	targetBackendURL := os.Getenv("TARGET_BACKEND_URL")
	if targetBackendURL == "" {
		targetBackendURL = "http://localhost:8080"
	}

	proxyListenAddr := os.Getenv("PROXY_LISTEN_ADDR")
	if proxyListenAddr == "" {
		proxyListenAddr = ":9090"
	}

	log.Printf("Starting reverse proxy on %s, forwarding to %s\n", proxyListenAddr, targetBackendURL)

	proxy, err := newProxyServer(targetBackendURL)
	if err != nil {
		log.Fatalf("Critical error creating proxy: %v", err)
	}

	server := &http.Server{
		Addr:    proxyListenAddr,
		Handler: proxy,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting proxy server: %v", err)
	}
}
