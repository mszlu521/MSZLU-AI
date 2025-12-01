package auths

import (
	"context"
	"model"

	"github.com/google/uuid"
	"github.com/mszlu521/thunder/gorms"
	"gorm.io/gorm"
)

type models struct {
	db *gorm.DB
}

func (m *models) findByUsernameOrEmail(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := m.db.WithContext(ctx).Where("username = ? or email = ?", username, username).First(&user).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &user, err
}

func (m *models) findById(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &user, err
}

func (m *models) updateUser(ctx context.Context, tx *gorm.DB, u *model.User) error {
	if tx == nil {
		tx = m.db
	}
	return tx.WithContext(ctx).Updates(u).Error
}

func (m *models) saveUser(ctx context.Context, tx *gorm.DB, user *model.User) error {
	if tx == nil {
		tx = m.db
	}
	return tx.WithContext(ctx).Create(user).Error
}

func (m *models) transaction(ctx context.Context, f func(tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(f)
}

func (m *models) findByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := m.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &user, err
}

func (m *models) findByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := m.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &user, err
}

func newModel(db *gorm.DB) *models {
	return &models{db: db}
}
