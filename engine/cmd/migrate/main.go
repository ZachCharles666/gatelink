// migrate runner — 内部使用，不对外暴露
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	dir := "internal/db/migrations"
	if len(os.Args) > 2 {
		dir = os.Args[2]
	}

	m, err := migrate.New("file://"+dir, dbURL)
	if err != nil {
		log.Fatalf("create migrate: %v", err)
	}
	defer m.Close()

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("migrate up: %v", err)
		}
		v, _, _ := m.Version()
		fmt.Printf("Migration UP done, version: %d\n", v)
	case "down":
		if err := m.Steps(-1); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		v, _, _ := m.Version()
		fmt.Printf("Migration DOWN done, version: %d\n", v)
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("version: %v", err)
		}
		fmt.Printf("version: %d, dirty: %v\n", v, dirty)
	default:
		log.Fatalf("unknown command: %s (use up/down/version)", cmd)
	}
}
