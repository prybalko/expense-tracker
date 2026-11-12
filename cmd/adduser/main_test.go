package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_success.db")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-user", "testuser", "-password", "secret", "-db", dbPath}
	err := run(args, stdin, stdout, stderr)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "User testuser created successfully")
}

func TestRun_DuplicateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_duplicate.db")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-user", "testuser", "-password", "secret", "-db", dbPath}

	err := run(args, stdin, stdout, stderr)
	require.NoError(t, err, "first run should succeed")

	stdout.Reset()
	stderr.Reset()
	err = run(args, stdin, stdout, stderr)
	require.Error(t, err, "expected error on duplicate user")
	assert.Contains(t, err.Error(), "already exists")
}

func TestRun_MissingUserFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-password", "secret"}
	err := run(args, stdin, stdout, stderr)
	require.Error(t, err, "expected error for missing user flag")
	assert.Contains(t, err.Error(), "missing required flags: user")

	assert.Contains(t, stdout.String(), "Usage:")
}

func TestRun_InteractivePassword(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_interactive.db")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	stdin := bytes.NewBufferString("interactive_secret\n")

	args := []string{"-user", "interactive_user", "-db", dbPath}
	err := run(args, stdin, stdout, stderr)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Password: ")
	assert.Contains(t, output, "User interactive_user created successfully")
}

func TestRun_InteractivePassword_Empty(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	stdin := bytes.NewBufferString("\n")

	args := []string{"-user", "empty_pass_user"}
	err := run(args, stdin, stdout, stderr)
	require.Error(t, err, "expected error for empty password")
	assert.Contains(t, err.Error(), "password cannot be empty")
}

func TestRun_EnvVarOverride(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_env.db")

	t.Setenv("DB_PATH", dbPath)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-user", "envuser", "-password", "secret"}
	err := run(args, stdin, stdout, stderr)
	require.NoError(t, err)

	assert.FileExists(t, dbPath)
}

func TestRun_InvalidDBPath(t *testing.T) {
	tmpDir := t.TempDir()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-user", "failuser", "-password", "secret", "-db", tmpDir}
	err := run(args, stdin, stdout, stderr)
	require.Error(t, err, "expected error for invalid db path")
	assert.Contains(t, err.Error(), "failed to open database")
}

func TestRun_InvalidFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-invalid"}
	err := run(args, stdin, stdout, stderr)
	require.Error(t, err, "expected error for invalid flag")
	assert.Contains(t, err.Error(), "flag provided but not defined")
}
