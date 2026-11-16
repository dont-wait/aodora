package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	authapp "github.com/your-org/aodora/server/internal/application/auth"
	"github.com/your-org/aodora/server/internal/config"
	"github.com/your-org/aodora/server/internal/infrastructure/database"
	mongorepo "github.com/your-org/aodora/server/internal/infrastructure/mongo"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	email := flag.String("email", "", "admin email")
	name := flag.String("name", "", "admin full name")
	phone := flag.String("phone", "", "admin phone")
	flag.Parse()
	if strings.TrimSpace(*email) == "" || strings.TrimSpace(*name) == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/admin -email owner@example.com -name \"Store Owner\"")
		os.Exit(2)
	}
	emailValue := authapp.NormalizeEmail(*email)
	if err := authapp.ValidateEmail(emailValue); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Fprint(os.Stderr, "Admin password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil || len(password) < 8 || len(password) > 72 {
		fmt.Fprintln(os.Stderr, "admin password must be between 8 and 72 characters")
		os.Exit(2)
	}

	cfg, err := config.LoadAllConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	db, err := database.OpenMongo(context.Background(), database.Config{URI: cfg.Mongo.URI, DB: cfg.Mongo.DB})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close(context.Background())
	repo := mongorepo.NewAuthRepository(db.DB)
	if err := repo.UpsertAdmin(context.Background(), emailValue, strings.TrimSpace(*name), strings.TrimSpace(*phone), string(hash)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("admin account ready:", emailValue)
}
