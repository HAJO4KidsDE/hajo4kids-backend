package main

import (
	"encoding/json"
	"log"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/config"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/database"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/handlers"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/middleware"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/models"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(
		&models.User{},
		&models.Ziel{},
		&models.Kategorie{},
		&models.Bild{},
		&models.Rating{},
		&models.Favorit{},
		&models.Vermarkter{},
		&models.Veranstaltung{},
		&models.Trip{},
		&models.KontaktMessage{},
		&models.StatisticsQuery{},
		&models.ShopKategorie{},
		&models.ShopItem{},
		&models.Post{},
		&models.PostComment{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "HAJO4Kids API v1.0",
		ServerHeader: "HAJO4Kids",
		// Enable JSON escaping for XSS prevention
		JSONEncoder: json.Marshal,
	})

	// Security middleware
	app.Use(middleware.SecurityHeaders())
	app.Use(middleware.RateLimiter())
	
	// Global middleware
	app.Use(middleware.CORSConfig([]string{"*"}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Setup routes
	setupRoutes(app, cfg)

	// Start server
	log.Printf("Server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
	if err := app.Listen(cfg.Server.Host + ":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(app *fiber.App, cfg *config.Config) {
	// API v1
	v1 := app.Group("/api/v1")

	// Auth rate limiter (stricter)
	authLimiter := middleware.RateLimiterAuth()

	// Public routes
	auth := v1.Group("/auth")
	auth.Post("/register", authLimiter, handlers.Register)
	auth.Post("/login", authLimiter, handlers.Login)
	auth.Get("/me", middleware.AuthMiddleware(database.DB, cfg), handlers.GetMe)

	// Ziel endpoints (public read)
	ziele := v1.Group("/ziele")
	ziele.Get("/", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetZiele)
	ziele.Get("/:id", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetZiel)
	ziele.Get("/:id/events", handlers.GetZielEvents)
	ziele.Get("/:id/ratings", handlers.GetZielRatings)
	ziele.Get("/place/:placeid", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetZielByPlaceID)

	// Kategorie endpoints
	kategorien := v1.Group("/kategorien")
	kategorien.Get("/", handlers.GetKategorien)
	kategorien.Get("/:id", handlers.GetKategorie)

	// Vermarkter endpoints (public read)
	vermarkter := v1.Group("/vermarkter")
	vermarkter.Get("/", handlers.GetVermarkter)
	vermarkter.Get("/:id", handlers.GetVermarkterByID)

	// Veranstaltung endpoints (public read)
	veranstaltungen := v1.Group("/veranstaltungen")
	veranstaltungen.Get("/", handlers.GetVeranstaltungen)
	veranstaltungen.Get("/:id", handlers.GetVeranstaltung)

	// Trip endpoints (public read, with ownership check)
	trips := v1.Group("/trips")
	trips.Get("/", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetTrips)
	trips.Get("/:id", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetTrip)

	// Search endpoint (public)
	v1.Get("/search", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.SearchZiele)

	// Kontakt form (public)
	v1.Post("/kontakt", authLimiter, handlers.SubmitKontakt)

	// Sitemap (public)
	v1.Get("/sitemap.xml", handlers.GetSitemap)

	// Bild endpoints (public read, auth for upload/delete)
	bilder := v1.Group("/bilder")
	bilder.Get("/", handlers.GetBilder)
	bilder.Get("/:id", handlers.GetBild)

	// Media files (public)
	media := app.Group("/media")
	media.Get("/bilder/:id", handlers.ServeBild)
	media.Get("/bilder/:id/thumb/:width/:height", handlers.ServeBildThumbnail)

	// Protected routes
	protected := v1.Use(middleware.AuthMiddleware(database.DB, cfg))

	// Ziel management (auth required)
	protected.Post("/ziele", handlers.CreateZiel)
	protected.Put("/ziele/:id", handlers.UpdateZiel)
	protected.Delete("/ziele/:id", handlers.DeleteZiel)

	// Kategorie management
	protected.Post("/kategorien", middleware.RoleMiddleware("admin", "reporter"), handlers.CreateKategorie)
	protected.Put("/kategorien/:id", middleware.RoleMiddleware("admin", "reporter"), handlers.UpdateKategorie)
	protected.Delete("/kategorien/:id", middleware.RoleMiddleware("admin"), handlers.DeleteKategorie)

	// Favoriten
	favoriten := protected.Group("/favoriten")
	favoriten.Get("/", handlers.GetFavoriten)
	favoriten.Post("/:zielid", handlers.AddFavorit)
	favoriten.Delete("/:zielid", handlers.RemoveFavorit)

	// Ratings
	ratings := protected.Group("/ratings")
	ratings.Post("/", handlers.CreateRating)
	ratings.Put("/:id", handlers.UpdateRating)
	ratings.Delete("/:id", handlers.DeleteRating)

	// Bild management (auth required)
	protected.Post("/bilder", handlers.UploadBild)
	protected.Put("/bilder/:id", handlers.UpdateBild)
	protected.Delete("/bilder/:id", handlers.DeleteBild)

	// Veranstaltungs management (auth required, reporter+)
	protected.Post("/veranstaltungen", middleware.RoleMiddleware("admin", "reporter"), handlers.CreateVeranstaltung)
	protected.Put("/veranstaltungen/:id", middleware.RoleMiddleware("admin", "reporter"), handlers.UpdateVeranstaltung)
	protected.Delete("/veranstaltungen/:id", middleware.RoleMiddleware("admin", "reporter"), handlers.DeleteVeranstaltung)

	// Vermarkter management (auth required, admin only)
	protected.Post("/vermarkter", middleware.RoleMiddleware("admin"), handlers.CreateVermarkter)
	protected.Put("/vermarkter/:id", middleware.RoleMiddleware("admin"), handlers.UpdateVermarkter)
	protected.Delete("/vermarkter/:id", middleware.RoleMiddleware("admin"), handlers.DeleteVermarkter)

	// Trip management (auth required)
	protected.Post("/trips", handlers.CreateTrip)
	protected.Put("/trips/:id", handlers.UpdateTrip)
	protected.Delete("/trips/:id", handlers.DeleteTrip)
	protected.Post("/trips/:id/ziele/:zielid", handlers.AddTripZiel)
	protected.Delete("/trips/:id/ziele/:zielid", handlers.RemoveTripZiel)

	// Shop kategorien (public read)
	shopKategorien := v1.Group("/shop/kategorien")
	shopKategorien.Get("/", handlers.GetShopKategorien)
	shopKategorien.Get("/:id", handlers.GetShopKategorie)

	// Shop items (public read)
	shopItems := v1.Group("/shop/items")
	shopItems.Get("/", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetShopItems)
	shopItems.Get("/:id", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetShopItem)
	shopItems.Get("/slug/:slug", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetShopItemBySlug)

	// Posts (public read)
	posts := v1.Group("/posts")
	posts.Get("/", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetPosts)
	posts.Get("/:id", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetPost)
	posts.Get("/slug/:slug", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.GetPostBySlug)
	posts.Get("/:id/comments", handlers.GetPostComments)
	posts.Post("/:id/comments", middleware.OptionalAuthMiddleware(database.DB, cfg), handlers.CreatePostComment)

	// Shop management (auth required, reporter+)
	protected.Post("/shop/kategorien", middleware.RoleMiddleware("admin", "reporter"), handlers.CreateShopKategorie)
	protected.Put("/shop/kategorien/:id", middleware.RoleMiddleware("admin", "reporter"), handlers.UpdateShopKategorie)
	protected.Delete("/shop/kategorien/:id", middleware.RoleMiddleware("admin"), handlers.DeleteShopKategorie)
	protected.Post("/shop/items", middleware.RoleMiddleware("admin", "reporter"), handlers.CreateShopItem)
	protected.Put("/shop/items/:id", middleware.RoleMiddleware("admin", "reporter"), handlers.UpdateShopItem)
	protected.Delete("/shop/items/:id", middleware.RoleMiddleware("admin"), handlers.DeleteShopItem)

	// Post management (auth required, reporter+)
	protected.Post("/posts", middleware.RoleMiddleware("admin", "reporter"), handlers.CreatePost)
	protected.Put("/posts/:id", handlers.UpdatePost)  // Owner or admin check in handler
	protected.Delete("/posts/:id", handlers.DeletePost)  // Owner or admin check in handler

	// Admin routes
	admin := v1.Group("/admin", middleware.AuthMiddleware(database.DB, cfg), middleware.RoleMiddleware("admin"))
	admin.Get("/users", handlers.GetUsers)
	admin.Put("/users/:id", handlers.UpdateUser)
	admin.Delete("/users/:id", handlers.DeleteUser)
	admin.Get("/kontakt", handlers.GetKontaktMessages)
	admin.Put("/kontakt/:id/read", handlers.MarkKontaktRead)
	admin.Get("/statistics", handlers.GetStatisticsOverview)
	admin.Get("/statistics/search", handlers.GetSearchStatistics)
	admin.Get("/comments", handlers.GetAllComments)
	admin.Put("/comments/:id/approve", handlers.ApproveComment)
	admin.Delete("/comments/:id", handlers.DeleteComment)
}