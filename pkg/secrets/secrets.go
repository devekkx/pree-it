package secrets

import (
	"fmt"
	"os"
	"strings"
)

// MustRead resolves a secret by name using the following priority:
//  1. File at the path given by env var <NAME>_FILE (Docker Secrets pattern)
//  2. File at /run/secrets/<name>  (Docker Swarm / Compose default mount)
//  3. Environment variable <NAME>  (local dev fallback only)
//
// Panics at startup if the secret cannot be resolved.
// This is intentional: a service that cannot read its secrets must not start.
func MustRead(name string) string {
	val, err := Read(name)
	if err != nil {
		panic(fmt.Sprintf("secrets.MustRead(%q): %v", name, err))
	}
	return val
}

// Read is the non-panicking variant.
func Read(name string) (string, error) {
	// 1. Explicit file path override via env
	if path := os.Getenv(strings.ToUpper(name) + "_FILE"); path != "" {
		return readFile(path, name)
	}

	// 2. Docker default mount path
	dockerPath := "/run/secrets/" + name
	if val, err := readFile(dockerPath, name); err == nil {
		return val, nil
	}

	// 3. Plain env var — dev only
	if val := os.Getenv(strings.ToUpper(name)); val != "" {
		return val, nil
	}

	return "", fmt.Errorf(
		"secret %q not found: checked ${%s_FILE}, %s, and ${%s}",
		name,
		strings.ToUpper(name),
		dockerPath,
		strings.ToUpper(name),
	)
}

func readFile(path, name string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read secret file %q for %q: %w", path, name, err)
	}
	val := strings.TrimSpace(string(data))
	if val == "" {
		return "", fmt.Errorf("secret file %q for %q is empty", path, name)
	}
	return val, nil
}
