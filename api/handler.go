package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/spf13/viper"
	"github.com/triapex/auth/api/middleware"
	"log"
	"net/http"
)

func usersRouter(ctrl *UsersController) http.Handler {
	h := chi.NewRouter()

	h.Group(func(r chi.Router) {
		allowedRolesForAuth := viper.GetStringSlice("auth.allowed_roles")
		r.Use(middleware.AuthMiddleware(ctrl.RSAPublicKey))
		r.Use(middleware.RoleAuthorizationMiddleware(allowedRolesForAuth))
		r.Post("/register", ctrl.SignUpUser)
		r.Post("/signup-register", ctrl.SignUpUser)
		r.Post("/register", ctrl.RegisterUser)
		r.Post("/enroll", ctrl.Enroll)
		r.Delete("/revoke", ctrl.Revoke)
	})

	h.Group(func(r chi.Router) {
		r.Get("/sso/login", ctrl.ssoLogin)
		r.Get("/sso/callback", ctrl.callback)
		r.Post("/login", ctrl.Login)
	})

	return h
}

func healthRouter(ctrl *SystemController) http.Handler {
	log.Println("healthRouter")
	h := chi.NewRouter()
	h.Group(func(r chi.Router) {
		// add all system check here, like: api, db connection, ......
		r.Get("/api", ctrl.apiCheck)
	})
	return h
}
