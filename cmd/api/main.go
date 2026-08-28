package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/proxy"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	godotenv.Load()

	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	viewService := services.NewViewService(db)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"Failed to load configuration: %v",
			err,
		)
	}

	valkeyClient, err := database.ConnectValkey(cfg)
	if err != nil {
		log.Fatalf(
			"Failed to connect to valkey: %v",
			err,
		)
	}

	// Servicio encargado de validar tokens
	authService := services.NewAuthService(
		valkeyClient,
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

		protectedUsers.GET(
			"/ipm-by-domain",
			handlers.ViewHandler(viewService, "vw_ipm_by_domain"),
		)
		protectedUsers.GET(
			"/average-deprivations",
			handlers.ViewHandler(viewService, "vw_average_deprivations"),
		)
		protectedUsers.GET(
			"/deprivations-by-variable",
			handlers.ViewHandler(viewService, "vw_deprivations_by_variable"),
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
