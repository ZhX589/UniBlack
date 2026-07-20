package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("UniBlack server starting...")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server running on port %s\n", port)
}
