package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
)

func main() {
	const indexPath = "web/dist/index.html"

	_, err := os.Stat(indexPath)
	if err == nil {
		return
	}
	if !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("check development frontend placeholder: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		log.Fatalf("create development frontend directory: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("<!doctype html><html><body></body></html>\n"), 0o644); err != nil {
		log.Fatalf("create development frontend placeholder: %v", err)
	}
}
