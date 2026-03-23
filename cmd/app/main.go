package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"wallet-service/internal/config"
	"wallet-service/internal/pg"
	"wallet-service/internal/wallet"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pg.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("cannot connect to db: %v", err)
	}
	defer pool.Close()

	repo := wallet.NewPostgresRepository(pool)
	handler := wallet.NewHandler(repo)

	mux := http.NewServeMux()
	handler.Register(mux)

	addr := ":" + cfg.AppPort
	log.Printf("server started on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
