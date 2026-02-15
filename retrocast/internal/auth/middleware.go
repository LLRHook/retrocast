package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// Middleware returns an Echo middleware that validates JWT access tokens.
// It extracts "Bearer <token>" from the Authorization header, validates it,
// and sets "user_id" in the Echo context.
func (ts *TokenService) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			token, found := strings.CutPrefix(header, "Bearer ")
			if !found || token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
			}

			claims, err := ts.ValidateAccessToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set("user_id", claims.UserID)
			return next(c)
		}
	}
}

// GetUserID extracts the authenticated user ID from the Echo context.
func GetUserID(c echo.Context) int64 {
	return c.Get("user_id").(int64)
}
