package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pkarpovich/tg-relay-bot/app/config"
	"log"
	"net/http"
)

type Server struct {
	config          *config.Config
	server          *http.Server
	messagesForSend chan string
}

func CreateServer(cfg *config.Config, messagesForSend chan string) *Server {
	mux := http.NewServeMux()
	server := &Server{
		config:          cfg,
		messagesForSend: messagesForSend,
	}

	mux.HandleFunc("GET /health", server.healthHandler)
	mux.HandleFunc("POST /send", server.sendHandler)
	mux.HandleFunc("POST /webhook", server.webhookHandler)

	server.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Http.Port),
		Handler: mux,
	}

	return server
}

func (s *Server) Start(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		log.Printf("[INFO] Starting HTTP server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

type HealthResponse struct {
	Ok bool `json:"ok"`
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(HealthResponse{Ok: true})
	if err != nil {
		log.Printf("[ERROR] Failed to write response: %s", err)
	}
}

func (s *Server) sendHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Header.Get("X-Secret") != s.config.Http.SecretApiKey {
		s.respondWithError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var data struct {
		Message string `json:"message"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		s.respondWithError(w, err, http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Sending message: %s", data.Message)
	s.messagesForSend <- data.Message

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(HealthResponse{Ok: true})
	if err != nil {
		log.Printf("[ERROR] Failed to write response: %s", err)
	}
}

func (s *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var data struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		s.respondWithError(w, err, http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Received webhook notification: %s", data.Content)
	s.messagesForSend <- data.Content

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(HealthResponse{Ok: true})
	if err != nil {
		log.Printf("[ERROR] Failed to write response: %s", err)
	}
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

func (s *Server) respondWithError(w http.ResponseWriter, err error, code int) {
	log.Printf("[ERROR] Internal server error: %s", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err = json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error(), Status: code})
	if err != nil {
		log.Printf("[ERROR] Failed to write response: %s", err)
	}
}
