package main

import (
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

	// Servicio para consumir la API de IA (predicciones)
	predictionService := services.NewPredictionService(cfg.AIService.BaseURL)

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

	viewHandler := handlers.NewViewHandler(viewService)
	predictionHandler := handlers.NewPredictionHandler(predictionService)

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
			"",
			handlers.ProxyHandler(userProxy),
		)

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
			handlers.ViewTotalHandler(viewService, "vw_ipm_by_domain"),
		)
		protectedUsers.GET(
			"/average-deprivations",
			handlers.ViewTotalHandler(viewService, "vw_average_deprivations"),
		)
		protectedUsers.GET(
			"/deprivations-by-variable",
			handlers.ViewTotalHandler(viewService, "vw_deprivations_by_variable"),
		)

		protectedUsers.GET(
			"/national-poverty",
			handlers.ViewTotalHandler(viewService, "vw_dashboard03_national_poverty"),
		)

		protectedUsers.GET(
			"/poverty-by-age",
			handlers.ViewTotalHandler(viewService, "vw_dashboard03_poverty_by_age"),
		)

		protectedUsers.GET(
			"/deprivations-contribution",
			handlers.ViewTotalHandler(viewService, "vw_dashboard03_deprivation_contribution"),
		)

		protectedUsers.GET(
			"/dimension-contribution",
			handlers.ViewTotalHandler(viewService, "vw_dimension_contribution"),
		)

		protectedUsers.GET(
			"/incidence-by-household-head-sex",
			handlers.ViewTotalHandler(viewService, "vw_incidence_by_household_head_sex"),
		)

		protectedUsers.GET(
			"/incidence-by-person-sex",
			handlers.ViewTotalHandler(viewService, "vw_incidence_by_person_sex"),
		)

		protectedUsers.POST(
			"/filtered",
			viewHandler.GetViewFilteredData,
		)

		// Historia: Filtrar por categoría IPM / Filtrar por ubicación
		// (obtiene los valores distintos de una columna para poblar los filtros)
		protectedUsers.GET(
			"/categories",
			viewHandler.GetCategories,
		)

		// Historia: Visualizar predicción (consume la API de IA)
		protectedUsers.POST(
			"/predictions/:type",
			predictionHandler.Predict,
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
