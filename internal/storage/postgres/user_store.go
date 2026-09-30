package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"particleapp/internal/models"
	"particleapp/internal/storage"
)

func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	err := s.db.WithContext(ctx).Create(user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return storage.ErrDuplicatedKey
	}
	return err
}

func (s *Store) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
