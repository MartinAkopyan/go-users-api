package http

import (
	"log/slog"
	"reflect"
	"strings"

	"github.com/MartinAkopyan/go-users-api/internal/service"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc      *service.UserService
	validate *validator.Validate
	logger   *slog.Logger
}

func NewHandler(svc *service.UserService, logger *slog.Logger) *Handler {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		return strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
	})
	return &Handler{svc: svc, validate: v, logger: logger}
}
