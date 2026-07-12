package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/rozy/backend/internal/platform/httpx"
)

// FlexibleMiddleware accepts JWT via Authorization header or ?token= query (for img tags).
func FlexibleMiddleware(tokens *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := r.URL.Query().Get("token")
			if tokenString == "" {
				header := r.Header.Get("Authorization")
				if strings.HasPrefix(header, "Bearer ") {
					tokenString = strings.TrimPrefix(header, "Bearer ")
				}
			}
			if tokenString == "" {
				httpx.Error(w, http.StatusUnauthorized, "missing token")
				return
			}
			claims, err := tokens.Parse(tokenString)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
