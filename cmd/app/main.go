package main

import (
	"context"
	"fmt"
	"os"

	r "github.com/poom90914/pulumi_golang/intenal/http"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

// define request/response types for various REST ops

func main() {
	// ensurePlugins()
	app := r.NewRouter()
	app.Run(":8080")
}

// ensure plugins runs once before the server boots up
// making sure the proper pulumi plugins are installed
func ensurePlugins() {
	ctx := context.Background()
	w, err := auto.NewLocalWorkspace(ctx)
	if err != nil {
		fmt.Printf("Failed to setup and run http server: %v\n", err)
		os.Exit(1)
	}
	err = w.InstallPlugin(ctx, "aws", "v3.2.1")
	if err != nil {
		fmt.Printf("Failed to install program plugins: %v\n", err)
		os.Exit(1)

	}
}
