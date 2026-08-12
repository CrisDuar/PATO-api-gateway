package main

import (
	"fmt"
	"log"

	"backend/internal/config"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/proxy"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"Failed to load configuration: %v",
			err,
		)
	}

	// Servicio encargado de validar tokens
	authService := services.NewAuthService(
		cfg.UserService.BaseURL,
	)

	// Proxy hacia user-service
	userProxy, err := proxy.NewReverseProxy(
		cfg.UserService.BaseURL,
	)

	if err != nil {
		log.Fatalf(
			"Failed to create user service proxy: %v",
			err,
		)
	}

	router := gin.Default()

	// Middlewares generales
	router.Use(
		middleware.RequestID(),
		middleware.CORS(
			cfg.CORS.AllowedOrigin,
		),
	)

	// Health del Gateway
	router.GET("/health", func(c *gin.Context) {

		c.JSON(200, gin.H{
			"status": "healthy",
			"app":    cfg.Server.Name,
		})
	})

	// =========================
	// RUTAS PÚBLICAS
	// =========================

	publicUsers := router.Group("/api/users")
	{
		publicUsers.POST(
			"/register",
			handlers.ProxyHandler(userProxy),
		)

		publicUsers.POST(
			"/login",
			handlers.ProxyHandler(userProxy),
		)

		publicUsers.POST(
			"/verify-email",
			handlers.ProxyHandler(userProxy),
		)

		publicUsers.POST(
			"/forgot-password",
			handlers.ProxyHandler(userProxy),
		)

		publicUsers.POST(
			"/reset-password",
			handlers.ProxyHandler(userProxy),
		)
	}

	// =========================
	// RUTAS PROTEGIDAS
	// =========================

	protectedUsers := router.Group("/api/users")

	protectedUsers.Use(
		middleware.AuthMiddleware(authService),
	)

	{
		protectedUsers.GET(
			"/me",
			handlers.ProxyHandler(userProxy),
		)

		protectedUsers.POST(
			"/logout",
			handlers.ProxyHandler(userProxy),
		)

		protectedUsers.PATCH(
			"/email",
			handlers.ProxyHandler(userProxy),
		)

		protectedUsers.PATCH(
			"/password",
			handlers.ProxyHandler(userProxy),
		)
	}

	addr := fmt.Sprintf(
		":%s",
		cfg.Server.Port,
	)

	log.Printf(
		"Starting %s on %s",
		cfg.Server.Name,
		addr,
	)

	if err := router.Run(addr); err != nil {
		log.Fatalf(
			"Failed to start server: %v",
			err,
		)
	}
}
