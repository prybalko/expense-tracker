package e2e

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// 1. Build the React bundle into web/dist so the embedded SPA handler
	//    serves the real UI (and not the placeholder index.html committed
	//    for //go:embed). Without this every selector misses.
	if err := ensureReactBundle(); err != nil {
		fmt.Printf("Failed to build React bundle: %v\n", err)
		return 1
	}

	// 2. Build the binary
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
	// Remove the WAL sidecars along with the DB — a stale -wal from a
	// previous run paired with a fresh DB file corrupts the new database.
	removeDB := func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}
	removeDB() // Ensure clean state
	defer removeDB()

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

// ensureReactBundle builds web/dist when it's missing or still the
// placeholder shipped for //go:embed. Skips work when a real bundle is
// already on disk so iterative test runs stay fast.
func ensureReactBundle() error {
	appDir := "../web/app"
	distDir := "../web/dist"
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		if _, err := os.Stat("web/app"); err == nil {
			appDir = "web/app"
			distDir = "web/dist"
		} else {
			return fmt.Errorf("could not locate web/app to build")
		}
	}

	if hasRealBundle(distDir) {
		return nil
	}

	// `npm ci` requires package-lock.json; fall back to `npm install` if it
	// isn't there (fresh checkout / out-of-band setup).
	installArgs := []string{"ci"}
	if _, err := os.Stat(filepath.Join(appDir, "package-lock.json")); os.IsNotExist(err) {
		installArgs = []string{"install"}
	}

	if _, err := os.Stat(filepath.Join(appDir, "node_modules")); os.IsNotExist(err) {
		install := exec.Command("npm", installArgs...)
		install.Dir = appDir
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("npm %s in %s: %w", installArgs[0], appDir, err)
		}
	}

	build := exec.Command("npm", "run", "build")
	build.Dir = appDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("npm run build in %s: %w", appDir, err)
	}
	if !hasRealBundle(distDir) {
		return fmt.Errorf("npm run build finished but %s/assets is empty", distDir)
	}
	return nil
}

// hasRealBundle returns true if dist/index.html is the bundle Vite emits
// (which references the hashed assets/index-*.js script). The placeholder
// committed for //go:embed has only a static <p> body and no script tag,
// so we check for the script reference rather than the assets directory —
// stale assets can survive a checkout that restores the placeholder.
func hasRealBundle(distDir string) bool {
	html, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		return false
	}
	return strings.Contains(string(html), "/assets/index-")
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
