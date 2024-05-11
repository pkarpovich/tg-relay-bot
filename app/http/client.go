package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pkarpovich/tg-relay-bot/app/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Client struct {
	Config          *config.Config
	MessagesForSend chan string
}

func CreateClient(cfg *config.Config, messagesForSend chan string) *Client {
	return &Client{
		MessagesForSend: messagesForSend,
		Config:          cfg,
	}
}

func (hc *Client) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", hc.healthHandler)
	mux.HandleFunc("POST /send", hc.sendHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", hc.Config.Http.Port),
		Handler: mux,
	}

	go func() {
		log.Printf("[INFO] Starting HTTP server on %s", server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[ERROR] HTTP server error: %s", err)
			return
		}
		log.Printf("[INFO] HTTP server stopped")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] HTTP server shutdown error: %s", err)
	}
	log.Printf("[INFO] HTTP server stopped")
}

type HealthResponse struct {
	Ok bool `json:"ok"`
}

func (hc *Client) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(HealthResponse{Ok: true})
	if err != nil {
		log.Printf("[ERROR] Failed to write response: %s", err)
	}
}

func (hc *Client) sendHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Header.Get("X-Secret") != hc.Config.Http.SecretApiKey {
		hc.respondWithError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var data struct {
		Message string `json:"message"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		hc.respondWithError(w, err, http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Sending message: %s", data.Message)
	hc.MessagesForSend <- data.Message

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(HealthResponse{Ok: true})
	if err != nil {
		log.Printf("[ERROR] Failed to write response: %s", err)
	}
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

func (hc *Client) respondWithError(w http.ResponseWriter, err error, code int) {
	log.Printf("[ERROR] Internal server error: %s", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err = json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error(), Status: code})
	if err != nil {
		log.Printf("[ERROR] Failed to write response: %s", err)
	}
}
