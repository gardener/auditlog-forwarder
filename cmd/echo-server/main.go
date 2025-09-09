package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var (
	logBody     = flag.Bool("log-body", false, "Should the request body be logged")
	tlsCertFile = flag.String("tls-cert-file", "", "Path to TLS certificate file (required for HTTPS)")
	tlsKeyFile  = flag.String("tls-key-file", "", "Path to TLS private key file (required for HTTPS)")
)

func main() {
	port := flag.Int("port", 8000, "Port to listen on")
	flag.Parse()

	srv := &http.Server{
		Handler:           http.HandlerFunc(genericHandler),
		Addr:              ":" + fmt.Sprint(*port),
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}

	if *tlsCertFile != "" && *tlsKeyFile != "" {
		log.Printf("Starting HTTPS echo server on port %d", *port)
		log.Fatal(srv.ListenAndServeTLS(*tlsCertFile, *tlsKeyFile))
	}

	log.Printf("Starting HTTP echo server on port %d", *port)
	log.Fatal(srv.ListenAndServe())
}

func genericHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		w.Header().Add("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status": "ok"}`))
		if err != nil {
			log.Print(err)
		}
		return
	}

	if r.Method == http.MethodPost {
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("Error closing body: %s\n", err)
			}
		}()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			http.Error(w, `{"success": false}`, http.StatusInternalServerError)
			return
		}
		if *logBody {
			log.Printf("Method: %s, Path: %s, Body: %s", r.Method, r.URL.Path, string(body))
		} else {
			log.Printf("Method: %s, Path: %s", r.Method, r.URL.Path)
		}
	} else {
		log.Printf("Method: %s, Path: %s", r.Method, r.URL.Path)
	}

	w.Header().Add("Content-Type", "application/json")
	_, err := w.Write([]byte(`{"success": true}`))
	if err != nil {
		log.Print(err)
	}
}
