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
	if err := ensureReactBundle(); err != nil {
		fmt.Printf("Failed to build React bundle: %v\n", err)
		return 1
	}

	buildPath := filepath.Join(os.TempDir(), "expense-tracker-test")

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

	dbPath = filepath.Join(os.TempDir(), "test_expenses.db")
	// A stale -wal from a previous run corrupts a fresh DB file.
	removeDB := func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}
	removeDB()
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
	serverCmd.Dir = ".."
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return 1
	}

	ready := waitForServer(appURL, 50, 100*time.Millisecond)
	if !ready {
		fmt.Println("Server failed to start or is not reachable")
		serverCmd.Process.Kill()
		return 1
	}

	code := m.Run()

	if err := serverCmd.Process.Kill(); err != nil {
		fmt.Printf("Failed to kill server: %v\n", err)
	}

	return code
}

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

// hasRealBundle checks index.html, not assets/: stale assets survive a checkout that restores the placeholder.
func hasRealBundle(distDir string) bool {
	html, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		return false
	}
	return strings.Contains(string(html), "/assets/index-")
}

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
