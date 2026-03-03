package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORSConfig returns a configured CORS middleware
func CORSConfig(allowedOrigins []string) fiber.Handler {
	// If wildcard, disable credentials for CORS security
	allowCredentials := true
	origins := strings.Join(allowedOrigins, ",")
	if origins == "*" {
		allowCredentials = false
	}
	
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
		AllowCredentials: allowCredentials,
		MaxAge:           86400,
	})
}