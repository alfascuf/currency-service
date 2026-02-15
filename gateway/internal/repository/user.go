package repository

import (
	"fmt"
	"sync"

	"github.com/alfascuf/PROD1/gateway/internal/models"
)

// UserRepository interface to work with user
type UserRepository interface {
	GetByLogin(login string) (*models.User, error)
	GetByID(id int) (*models.User, error)
}

// inMemoryUserRepository - storage for users
type inMemoryUserRepository struct {
	users map[string]*models.User // Login + user
	mu    sync.RWMutex            // anti race
}

// NewUserRepository create new rep with test users
func NewUserRepository() UserRepository {
	// Создаём несколько тестовых пользователей
	users := map[string]*models.User{
		"user1": {
			ID:       1,
			Login:    "user1",
			Password: "password123", // В реальности - bcrypt хеш!
		},
		"admin": {
			ID:       2,
			Login:    "admin",
			Password: "admin123",
		},
		"test": {
			ID:       3,
			Login:    "test",
			Password: "test123",
		},
	}
	return &inMemoryUserRepository{
		users: users,
	}
}

// GetByLogin finds user by login
func (r *inMemoryUserRepository) GetByLogin(login string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[login]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", login)
	}
	return user, nil
}

// GetByID finds user by ID
func (r *inMemoryUserRepository) GetByID(id int) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found: %d", id)
}
