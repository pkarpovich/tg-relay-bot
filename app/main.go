package main

import (
	"github.com/pkarpovich/tg-relay-bot/app/config"
	"log"
)

func main() {
	log.Printf("[INFO] Starting application...")

	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("[ERROR] Error reading config: %s", err)
	}

	run(cfg)
}

func run(cfg *config.Config) {
	log.Printf("[INFO] Running application with config: %+v", cfg)
}
