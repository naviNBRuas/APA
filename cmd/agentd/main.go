package main

import (
  "fmt"
  "net/http"
  "os"
)

func main() {
  // simple HTTP server to show the container is alive
  http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("apa: ok"))
  })
  port := "8080"
  if p := os.Getenv("APA_PORT"); p != "" {
    port = p
  }
  fmt.Println("APA agent starting on :" + port)
  http.ListenAndServe(":"+port, nil)
}
