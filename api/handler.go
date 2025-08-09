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
		allowedRolesForAuth := viper.GetStringSlice("enroller.allowed_roles")
		r.Use(middleware.AuthMiddleware(ctrl.RSAPublicKey))
		r.Use(middleware.RoleAuthorizationMiddleware(allowedRolesForAuth))
		r.Post("/signup", ctrl.SignUpUser)
		r.Post("/register", ctrl.RegisterUser)
		r.Post("/enroll", ctrl.Enroll)
		r.Post("/register-enroll", ctrl.RegisterAndEnrollUser)
		r.Delete("/revoke/{username}", ctrl.Revoke)
		r.Get("/msp/{username}", ctrl.GetMsp)
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
