package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pkarpovich/tg-relay-bot/app/config"
	"github.com/pkarpovich/tg-relay-bot/app/events"
)

type Server struct {
	config          *config.Config
	server          *http.Server
	messagesForSend chan events.MessagePayload
}

func CreateServer(cfg *config.Config, messagesForSend chan events.MessagePayload) *Server {
	mux := http.NewServeMux()
	server := &Server{
		config:          cfg,
		messagesForSend: messagesForSend,
	}

	mux.HandleFunc("GET /health", server.healthHandler)
	mux.HandleFunc("POST /send", server.sendHandler)
	mux.HandleFunc("POST /webhook", server.webhookHandler)

	server.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Http.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
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
		return fmt.Errorf("http server context done: %w", ctx.Err())
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}

type HealthResponse struct {
	Ok bool `json:"ok"`
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
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
		Message   string `json:"message"`
		ParseMode string `json:"parse_mode"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		s.respondWithError(w, err, http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Sending message: %s", data.Message)
	s.messagesForSend <- events.MessagePayload{Text: data.Message, ParseMode: data.ParseMode}

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
	s.messagesForSend <- events.MessagePayload{Text: data.Content}

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
