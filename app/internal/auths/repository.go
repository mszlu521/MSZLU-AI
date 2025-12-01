package auths

import (
	"context"
	"model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository interface {
	findByUsername(ctx context.Context, username string) (*model.User, error)
	findByEmail(ctx context.Context, email string) (*model.User, error)
	transaction(ctx context.Context, f func(tx *gorm.DB) error) error
	saveUser(ctx context.Context, tx *gorm.DB, m *model.User) error
	findById(ctx context.Context, id uuid.UUID) (*model.User, error)
	updateUser(ctx context.Context, tx *gorm.DB, u *model.User) error
	findByUsernameOrEmail(ctx context.Context, username string) (*model.User, error)
}
