# Go Proxy

A simple HTTP reverse proxy with backend health checking and a sample backend server, written in Go.

## Overview

This project consists of two main components:

1.  **Reverse Proxy (`cmd/reverse_proxy`)**: A load-balancing reverse proxy that distributes incoming HTTP requests across multiple backend servers. It performs periodic health checks on the backends and only forwards requests to healthy instances.
2.  **Backend Server (`cmd/backend_server`)**: A sample HTTP server that can be used as a target for the reverse proxy. It has a health check endpoint and a simple handler that echoes request information.

## Features

### Reverse Proxy

* **Load Balancing**: Distributes traffic to backend servers using a round-robin strategy.
* **Health Checking**: Periodically checks the health of backend servers by calling a configurable health endpoint (default `/health`). Unhealthy servers are temporarily removed from the load balancing pool.
* **Dynamic Backend Configuration**: Backend server URLs are configured via an environment variable.
* **Customizable Listen Address**: The proxy's listening address can be configured.
* **Error Handling**: Returns `503 Service Unavailable` if no healthy backends are available and `502 Bad Gateway` if communication with a selected backend fails.
* **Logging**: Provides logs for incoming requests, forwarding decisions, health check status, and errors.

### Backend Server

* **Health Check Endpoint**: Provides a `/health` endpoint that returns `HTTP 200 OK` when the server is healthy.
* **Request Echo**: The root path (`/`) handler displays information about the received request, including headers, the path, and the backend server's hostname and listening address.
* **Customizable Listen Address**: The backend server's listening address can be configured.

## How to Run

### Prerequisites

* Go installed on your system.

### 1. Backend Server

The backend server listens for HTTP requests.

**Environment Variables:**

* `BACKEND_LISTEN_ADDR`: The address and port for the backend server to listen on. Defaults to `:8080`.

**Running:**

```bash
# Navigate to the backend server directory
cd cmd/backend_server

# Set environment variable (optional, defaults to :8080)
export BACKEND_LISTEN_ADDR=":8081"

# Run the backend server
go run main.go
# Output: 🚀 starting backend server on :8081 (reporting as 8081)
```

Now, requests made to the reverse proxy (e.g., http://localhost:9000/some/path) will be forwarded to one of the healthy backend servers listed in TARGET_BACKEND_URLS.

How It Works (Reverse Proxy)
Initialization:

The proxy parses the TARGET_BACKEND_URLS environment variable to get a list of backend server addresses.
For each backend, it creates a Backend object which includes its URL and an httputil.ReverseProxy instance.
A server pool (ServerPool) is created to manage these backends.
Health Checking:

A background goroutine periodically sends GET requests to the /health endpoint of each backend server (defined by HEALTH_CHECK_INTERVAL).
If a backend returns HTTP 200 OK, it's marked as Alive. Otherwise, it's marked as not Alive.
Logs indicate changes in backend liveness.
Request Handling:

When a request arrives at the proxy:
The ServerPool attempts to get the next healthy backend using a round-robin approach (GetNextHealthyBackend).
If no healthy backend is found, the proxy responds with 503 Service Unavailable.
Otherwise, the request is forwarded to the selected healthy backend using its httputil.ReverseProxy instance.
The Host header is appropriately set for the backend.
If an error occurs while communicating with the backend (e.g., connection refused), the backend is marked as not Alive, and the proxy returns 502 Bad Gateway.
To-Do / Potential Enhancements
More sophisticated load-balancing algorithms (e.g., least connections, weighted round-robin).
Configuration via a file instead of only environment variables.
HTTPS support for both frontend and backend connections.
Graceful shutdown.
More detailed metrics and statistics.
