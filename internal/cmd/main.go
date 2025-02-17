package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wisp167/Shop/internal/server"
)

func main() {
	var err error
	app, err := server.SetupApplication()
	if err != nil {
		panic(err)
	}

	// Start the server in a goroutine
	go func() {
		if err := app.Start(); err != nil {
			panic(err)
		}
	}()

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
	log.Println("Server exiting")
	// Stop the server
	if err := app.Stop(); err != nil {
		panic(err)
	}
}
