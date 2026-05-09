package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {
	godotenv.Load()

	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		log.Fatal("PG_DSN not set")
	}

	// goose работает через database/sql — используем stdlib-обёртку pgx
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Fatal("parse dsn: ", err)
	}

	db := stdlib.OpenDB(*cfg)
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	// аргумент командной строки: up / down / status / reset
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if err := goose.RunContext(context.Background(), command, db, "migrations"); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}
}
