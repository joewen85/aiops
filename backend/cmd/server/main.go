package main

import (
	"log"

	"devops-system/backend/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("start app failed: %v", err)
	}
	addr := ":" + application.Config.AppPort
	log.Printf("server listening on %s", addr)
	if err := application.Engine.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
