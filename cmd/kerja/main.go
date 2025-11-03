package main

import (
	"context"

	"github.com/faizmokh/kerja/internal/cli"
)

func main() {
	ctx := context.Background()
	cli.Main(ctx)
}

