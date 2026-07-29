package middleware

import (
	"net/http"

	"golang-user-crud-assesment/internal/response"
)

func BasicAuth(username, password string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providedUsername, providedPassword, ok := r.BasicAuth()
		if !ok || providedUsername != username || providedPassword != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			response.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
