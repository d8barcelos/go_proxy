package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	listenAddressForResponse string
)

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK from %s", listenAddressForResponse)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()

	log.Printf("Backend (%s) at %s received request: %s %s from %s\n",
		hostname, listenAddressForResponse, r.Method, r.URL.Path, r.RemoteAddr)

	fmt.Fprintf(w, "Hello from Backend (%s) listening at %s!\n", hostname, listenAddressForResponse)
	fmt.Fprintf(w, "You've requested: %s\n\n", r.URL.Path)

	fmt.Fprintf(w, "Headers:\n")
	for name, headers := range r.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func main() {
	backendListenAddr := os.Getenv("BACKEND_LISTEN_ADDR")
	if backendListenAddr == "" {
		backendListenAddr = ":8080"
	}
	listenAddressForResponse = strings.TrimPrefix(backendListenAddr, ":")

	log.Printf("🚀 starting backend server on %s (reporting as %s)\n", backendListenAddr, listenAddressForResponse)

	http.HandleFunc("/health", healthCheckHandler)
	http.HandleFunc("/", helloHandler)

	if err := http.ListenAndServe(backendListenAddr, nil); err != nil {
		log.Fatalf("Error starting backend server: %v", err)
	}
}
