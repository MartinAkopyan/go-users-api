package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MartinAkopyan/go-users-api/internal/domain"
	"github.com/MartinAkopyan/go-users-api/internal/service"
)

type fakeUserRepo struct {
	users map[int64]domain.User
}

func (f *fakeUserRepo) Create(ctx context.Context, user domain.User) (domain.User, error) {
	return domain.User{}, nil
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id int64) (domain.User, error) {
	if user, ok := f.users[id]; ok {
		return user, nil
	}

	return domain.User{}, domain.ErrNotFound
}

func newTestRouter(repo *fakeUserRepo) http.Handler {
	svc := service.NewUserService(repo)
	h := NewHandler(svc, slog.New(slog.DiscardHandler))
	return NewRouter(h)
}

func TestGetUser_Success(t *testing.T) {
	repo := &fakeUserRepo{users: map[int64]domain.User{
		1: {ID: 1, Email: "ann@mail.com", PasswordHash: "secret-hash", Role: domain.RoleUser}},
	}

	router := newTestRouter(repo)
	var resp userResponse
	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}

	if strings.Contains(rec.Body.String(), "secret-hash") {
		t.Errorf("response contains password hash: %s", rec.Body.String())
	}

	err := json.NewDecoder(rec.Body).Decode(&resp)

	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Email != "ann@mail.com" {
		t.Errorf("got %s, want %s", resp.Email, "ann@mail.com")
	}
}

func TestGetUser_NotFound(t *testing.T) {
	repo := &fakeUserRepo{users: map[int64]domain.User{}}
	router := newTestRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetUser_InvalidID(t *testing.T) {
	repo := &fakeUserRepo{users: map[int64]domain.User{}}
	router := newTestRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
