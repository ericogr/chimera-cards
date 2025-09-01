package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/")
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()
	// Consider any status < 500 as healthy (including 404 for missing root)
	if resp.StatusCode >= 500 {
		os.Exit(1)
	}
	os.Exit(0)
}
