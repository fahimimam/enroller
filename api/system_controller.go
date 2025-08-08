package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/triapex/auth/api/response"
	"github.com/triapex/auth/internal/infra"
)

type SystemController struct {
	db infra.DB
}

func NewSystemController(db infra.DB) *SystemController {
	return &SystemController{
		db: db,
	}
}

func (s *SystemController) apiCheck(w http.ResponseWriter, r *http.Request) {
	log.Println("apiCheck")
	if err := s.connCheck(); err != nil {
		_ = response.ServeJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	response.ServeJSONData(w, "ok", http.StatusOK)
	return
}

func (s *SystemController) connCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	log.Println("db ping")
	if err := s.db.Ping(ctx); err != nil {
		return fmt.Errorf("postgres conn error: %v", err)
	}
	log.Println("kv ping")
	return nil
}
