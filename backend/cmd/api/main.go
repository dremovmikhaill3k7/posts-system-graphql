package main

import (
	"context"
	"log"
	"posts_service/internal/app"
)

func main() {
	ctx := context.Background()

	appl, err := app.New(ctx)
	if err != nil {
		log.Fatalf("ошибка настройки приложения %v", err)
	}

	appl.Run(ctx)
}
