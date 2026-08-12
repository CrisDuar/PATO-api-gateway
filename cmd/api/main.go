package main

import (
	"context"
	"fmt"
	"log"

	"backend/internal/config"
	"backend/internal/database"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	valkeyClient, err := database.ConnectValkey(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to valkey: %v", err)
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		valkeyStatus := "healthy"
		if err := valkeyClient.Ping(context.Background()).Err(); err != nil {
			valkeyStatus = "unhealthy"
		}

		c.JSON(200, gin.H{
			"status": "healthy",
			"app":    cfg.Server.Name,
			"valkey": valkeyStatus,
		})
	})

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Starting %s on %s", cfg.Server.Name, addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
