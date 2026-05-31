package middleware

import (
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/labstack/echo/v4"
	"github.com/shah-dhwanil/tasker/internal/errors"
)


const userContextKey = "user"

func ClerkAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return errors.NewUnauthorizedError(nil,"Authorization header is required", nil, nil)
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				return errors.NewUnauthorizedError(nil,"Invalid Authorization header format", nil, nil)
			}

			claims, err := jwt.Verify(c.Request().Context(), &jwt.VerifyParams{
				Token: token,
			})
			if err != nil {
				return errors.NewUnauthorizedError(err,"Invalid or expired token", nil, nil)
			}
			c.Set(userContextKey, claims)
			return next(c)
		}
	}
}


func GetUserFromContext(c echo.Context) (*clerk.SessionClaims, bool) {
	user, ok := c.Get(userContextKey).(*clerk.SessionClaims)
	return user, ok
}