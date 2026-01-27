package main

import (
	"log"
	"os"
	"path/filepath"

	"xiaoheiplay/internal/pkg/docs"
)

func main() {
	base := filepath.Join("docs")
	if err := os.MkdirAll(base, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "openapi.yaml"), []byte(docs.OpenAPIYAML), 0o644); err != nil {
		log.Fatalf("write openapi: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "api.md"), []byte(docs.APIMarkdownFile), 0o644); err != nil {
		log.Fatalf("write api: %v", err)
	}
	log.Printf("docs generated")
}
