package main

import (
	"context"
	"volo/routes"
)

func main() {
	ctx, _ := context.WithCancel(context.Background())
	routes.Run(ctx)
}
