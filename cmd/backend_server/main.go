package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()

	log.Printf("Backend (%s) received request: %s %s de %s\n", hostname, r.Method, r.URL.Path, r.RemoteAddr)

	fmt.Fprintf(w, "Hello from Backend (%s)!\n", hostname)
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

	log.Printf("🚀 starting server on %s\n", backendListenAddr)

	http.HandleFunc("/", helloHandler)

	if err := http.ListenAndServe(backendListenAddr, nil); err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}
