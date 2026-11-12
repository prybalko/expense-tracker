package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/storage"

	"golang.org/x/term"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("adduser", flag.ContinueOnError)
	fs.SetOutput(stderr)

	username := fs.String("user", "", "Username")
	passwordFlag := fs.String("password", "", "Password (optional, will prompt if omitted)")
	dbPath := fs.String("db", "expenses.db", "Path to database file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" {
		fmt.Fprintln(stdout, "Usage: adduser -user <username> [-password <password>] [-db <db_path>]")
		fs.PrintDefaults()
		return fmt.Errorf("missing required flags: user")
	}

	password := *passwordFlag
	if password == "" {
		fmt.Fprint(stdout, "Password: ")
		var err error
		password, err = readPassword(stdin)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Fprintln(stdout)
	}

	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if path := os.Getenv("DB_PATH"); path != "" && *dbPath == "expenses.db" {
		*dbPath = path
	}

	db, err := storage.NewDB(*dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	existingUser, err := db.GetUserByUsername(*username)
	if err == nil && existingUser != nil {
		return fmt.Errorf("user %s already exists", *username)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := db.CreateUser(*username, hash)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Fprintf(stdout, "User %s created successfully with ID %d\n", user.Username, user.ID)
	return nil
}

func readPassword(stdin io.Reader) (string, error) {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		bytePassword, err := term.ReadPassword(int(f.Fd()))
		if err != nil {
			return "", err
		}
		return string(bytePassword), nil
	}

	scanner := bufio.NewScanner(stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}
