package main

import (
	"log"
	"net/http"

	"github.com/YehhiAleksandra/oracle-gateway/internal/config"
	"github.com/YehhiAleksandra/oracle-gateway/internal/handlers"
)

func main() {
	cfg := config.Load()
	srv := handlers.New(cfg)
	log.Printf("oracle-gateway listening on %s models=%v", cfg.Addr, cfg.Models())
	if err := http.ListenAndServe(cfg.Addr, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}
