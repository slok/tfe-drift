package main

import (
	"context"
	"fmt"
	"os"
)

func run(ctx context.Context) error {
	return nil
}

func main() {
	ctx := context.Background()

	err := run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
}
