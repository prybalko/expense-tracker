package e2e

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var (
	appURL string
	dbPath string
)

func TestMain(m *testing.M) {
	os.Exit(runTestMain(m))
}

func runTestMain(m *testing.M) int {
	// 1. Build the binary
	buildPath := filepath.Join(os.TempDir(), "expense-tracker-test")
	
	// Determine correct path to cmd/server
	serverPath := "../cmd/server"
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		if _, err := os.Stat("cmd/server"); err == nil {
			serverPath = "./cmd/server"
		} else {
			fmt.Println("Could not find cmd/server to build")
			return 1
		}
	}

	cmd := exec.Command("go", "build", "-o", buildPath, serverPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to build app: %v\n%s\n", err, output)
		return 1
	}
	defer os.Remove(buildPath)

	// 2. Start the server
	dbPath = filepath.Join(os.TempDir(), "test_expenses.db")
	os.Remove(dbPath) // Ensure clean state
	defer os.Remove(dbPath)

	port := "8081"
	appURL = "http://localhost:" + port

	serverCmd := exec.Command(buildPath)
	serverCmd.Env = append(os.Environ(),
		"PORT="+port,
		"DB_PATH="+dbPath,
		"ADMIN_USER=testuser",
		"ADMIN_PASSWORD=testpass123",
	)
	serverCmd.Dir = ".." // Run from project root so it finds web/templates
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return 1
	}

	// Wait for server to be ready
	ready := waitForServer(appURL, 50, 100*time.Millisecond)
	if !ready {
		fmt.Println("Server failed to start or is not reachable")
		serverCmd.Process.Kill()
		return 1
	}

	// 3. Run tests
	code := m.Run()

	// 4. Cleanup
	if err := serverCmd.Process.Kill(); err != nil {
		fmt.Printf("Failed to kill server: %v\n", err)
	}

	return code
}

// waitForServer waits for the server to become ready
func waitForServer(url string, maxAttempts int, interval time.Duration) bool {
	for range maxAttempts {
		time.Sleep(interval)
		resp, err := http.Get(url + "/expenses")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 302 {
				return true
			}
		}
	}
	return false
}
