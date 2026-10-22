package service

import (
	"context"
	"errors"
	"testing"

	"github.com/MartinAkopyan/go-users-api/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepo struct {
	users map[string]domain.User
}

func (r *fakeUserRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {
	if _, ok := r.users[u.Email]; ok {
		return domain.User{}, domain.ErrAlreadyExists
	}
	id := int64(len(r.users)) + 1
	u.ID = id
	r.users[u.Email] = u

	return u, nil
}

func TestRegister_Success(t *testing.T) {
	repo := &fakeUserRepo{users: make(map[string]domain.User)}
	srv := NewUserService(repo)
	usr, err := srv.Register(context.Background(), "Full name", " Test@email.com  ", "password1")

	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if usr.ID == 0 {
		t.Errorf("ID = %d, want non-zero", usr.ID)
	}

	if usr.Email != "test@email.com" {
		t.Errorf("Email = %q, want %q", usr.Email, "test@email.com")
	}

	if usr.Role != domain.RoleUser {
		t.Errorf("Role = %q, want %q", usr.Role, domain.RoleUser)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte("password1")); err != nil {
		t.Errorf("hash doesn't match password: %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &fakeUserRepo{users: make(map[string]domain.User)}
	srv := NewUserService(repo)

	_, err := srv.Register(context.Background(), "Full name", "test@mail.com", "password1")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = srv.Register(context.Background(), "Full Name", "test@mail.com", "password1")

	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Errorf("err = %v, want %v", err, domain.ErrAlreadyExists)
	}
}
