//go:build ignore

package main

import (
	"fmt"
	"time"

	"github.com/naviNBRuas/APA/pkg/platform"
)

// Platform profile demo prints build-time feature toggles.
func main() {
	prof := platform.Current()
	fmt.Println("APA Platform Profile Demo")
	fmt.Println("========================")
	fmt.Printf("Minimal build: %v\n", prof.Minimal)
	fmt.Printf("TinyGo build:  %v\n", prof.TinyGo)
	fmt.Printf("Timestamp:     %s\n", time.Now().Format(time.RFC3339))
	fmt.Println("Use -tags minimal or -tags tinygo to flip these flags during builds.")
}
