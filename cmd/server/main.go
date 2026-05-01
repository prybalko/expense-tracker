package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"expense-tracker/internal/api"
	"expense-tracker/internal/auth"
	"expense-tracker/internal/storage"
	"expense-tracker/web"
)

// bootstrapUser creates a default user if none exist and credentials are provided via env vars.
func bootstrapUser(db *storage.DB) {
	count, err := db.UserCount()
	if err != nil {
		log.Printf("Warning: could not check user count: %v", err)
		return
	}

	if count > 0 {
		return // Users already exist
	}

	username := os.Getenv("ADMIN_USER")
	password := os.Getenv("ADMIN_PASSWORD")

	if username == "" || password == "" {
		username = "admin"
		var err error
		password, err = auth.GenerateRandomPassword()
		if err != nil {
			log.Printf("Failed to generate random password: %v", err)
			return
		}
		log.Println("=======================================================")
		log.Println("WARNING: Creating default admin user with random password")
		log.Printf("Password: %s", password)
		log.Println("=======================================================")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return
	}

	if _, err := db.CreateUser(username, hash); err != nil {
		log.Printf("Failed to create admin user: %v", err)
		return
	}

	log.Printf("Created admin user: %s", username)
}

// newSPAHandler returns a handler that serves files from the embedded
// dist FS. Requests for paths that don't match an embedded file fall
// back to index.html so the React Router can take over on the client.
func newSPAHandler(distFS embed.FS) (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}

	indexBytes, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(sub))

	serveIndex := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexBytes)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			serveIndex(w)
			return
		}

		f, err := sub.Open(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				serveIndex(w)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = f.Close()

		fileServer.ServeHTTP(w, r)
	}), nil
}

func main() {
	spaHandler, err := newSPAHandler(web.DistFS)
	if err != nil {
		log.Fatalf("Failed to initialize static handler: %v", err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "expenses.db"
	}

	db, err := storage.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	bootstrapUser(db)

	secureCookie := os.Getenv("SECURE_COOKIE") == "true"

	apiHandler := api.NewRouter(db, secureCookie)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	mux.Handle("/", spaHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	srv := &http.Server{
		Addr:              port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server starting on %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Error starting server: %v", err)
		return

	case <-shutdown:
		log.Println("Starting shutdown...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Could not stop server gracefully: %v", err)
			if err = srv.Close(); err != nil {
				log.Printf("Could not stop http server: %v", err)
			}
		}
		log.Println("Server stopped")
	}
}
