package main

import (
	"net/http"

	"github.com/justinas/nosurf"
)

func (a *application) LoadSession(next http.Handler) http.Handler {
	return a.session.LoadAndSave(next)
}

func (a *application) authRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId := a.session.GetInt(r.Context(), sessionKeyUserId)
		if userId == 0 {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		w.Header().Add("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	}
}

func (a *application) CSRFTokenRequired(next http.Handler) http.Handler {
	handler := nosurf.New(next)

	return handler
}

// func (a *application) recoverPanic(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		defer func() {
// 			if err := recover(); err != nil {
// 				w.Header().Set("announce", "Connection: Close")
// 				http.Error(w, "Internal server error", http.StatusInternalServerError)
// 			}
// 		}()

// 		next.ServeHTTP(w, r)
// 	})
// }
