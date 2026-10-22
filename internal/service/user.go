// Package service содержит бизнес-логику. Не знает ни про HTTP, ни про Postgres,
// с хранилищем общается только через интерфейс UserRepository.
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/MartinAkopyan/go-users-api/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository описывает, что сервису нужно от хранилища. Интерфейс объявлен здесь,
// у потребителя, и содержит только используемые методы. postgres.UserRepo
// подходит под него автоматически (совпадают методы), а в тестах вместо него
// подставляется fake без базы. Новые методы добавляются по мере надобности.
type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
}

// UserService держит зависимости. Поле типа интерфейса: сервису всё равно,
// какая реализация пришла.
type UserService struct {
	repo UserRepository
}

// NewUserService это конструктор. Конкретный репозиторий передаётся снаружи (из main),
// это ручной аналог DI-контейнера Laravel.
func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Register создаёт обычного пользователя. Данные уже провалидированы хендлером
// (формат email, длина полей), здесь только бизнес-правила.
func (s *UserService) Register(ctx context.Context, fullName, email, password string) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		// %w оборачивает исходную ошибку: в логе видно и шаг, и причину.
		return domain.User{}, fmt.Errorf("hashing password: %w", err)
	}

	u := domain.User{FullName: fullName, Email: email, PasswordHash: string(hash), Role: domain.RoleUser}

	usr, err := s.repo.Create(ctx, u)

	if err != nil {
		// Обёртка через %w сохраняет domain.ErrAlreadyExists внутри: хендлер
		// распознает её через errors.Is и ответит 409. С %v или errors.New
		// ошибка потерялась бы, и вышло бы 500.
		return domain.User{}, fmt.Errorf("creating user: %w", err)
	}

	return usr, nil
}
