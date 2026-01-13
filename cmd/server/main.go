package main

import (
	"context"
	"expense-tracker/internal/auth"
	"expense-tracker/internal/handlers"
	"expense-tracker/internal/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func setupRouter(h *handlers.Handlers, staticDir string) *http.ServeMux {
	mux := http.NewServeMux()

	// Static files (public)
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Auth routes (public)
	mux.HandleFunc("GET /login", h.LoginForm)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("GET /logout", h.Logout)

	// Root redirect
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/expenses", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// Protected routes (require authentication)
	mux.Handle("GET /expenses", h.AuthMiddleware(http.HandlerFunc(h.ListExpenses)))
	mux.Handle("GET /expenses/create", h.AuthMiddleware(http.HandlerFunc(h.CreateExpenseForm)))
	mux.Handle("POST /expenses", h.AuthMiddleware(http.HandlerFunc(h.CreateExpense)))
	mux.Handle("GET /expenses/{id}/edit", h.AuthMiddleware(http.HandlerFunc(h.EditExpenseForm)))
	mux.Handle("POST /expenses/{id}", h.AuthMiddleware(http.HandlerFunc(h.UpdateExpense)))
	mux.Handle("DELETE /expenses/{id}", h.AuthMiddleware(http.HandlerFunc(h.DeleteExpense)))
	mux.Handle("GET /statistics", h.AuthMiddleware(http.HandlerFunc(h.Statistics)))

	return mux
}

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
		// Generate default admin with random password
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

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "expenses.db"
	}

	db, err := storage.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create initial user if needed
	bootstrapUser(db)

	// Use secure cookies when running with HTTPS (production)
	secureCookie := os.Getenv("SECURE_COOKIE") == "true"

	h := handlers.NewHandlers(db, "web/templates", secureCookie)
	mux := setupRouter(h, "web/static")

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

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("API server starting on %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Channel to listen for interrupt or terminate signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Error starting server: %v", err)
		return

	case <-shutdown:
		log.Println("Starting shutdown...")

		// Create a context with a timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Could not stop server gracefully: %v", err)
			if err = srv.Close(); err != nil {
				log.Printf("Could not stop http server: %v", err)
			}
		}
		log.Println("Server stopped")
	}
}
