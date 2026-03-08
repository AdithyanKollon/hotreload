// testserver is a simple HTTP server used to demonstrate hotreload.
// Edit the message constant below and save — hotreload will rebuild
// and restart the server automatically.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// ✏️  Edit this message and save the file to trigger a hot reload!
const message = "Test Message"

const port = "8080"

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request received", "method", r.Method, "path", r.URL.Path)
		fmt.Fprintln(w, message)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("/time", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, time.Now().Format(time.RFC3339))
	})

	addr := ":" + port
	slog.Info("testserver listening", "addr", addr, "message", message)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
