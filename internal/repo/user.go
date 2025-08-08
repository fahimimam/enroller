package repo

import (
	"context"
	"github.com/triapex/auth/config"
	"github.com/triapex/auth/internal/infra"
	"github.com/triapex/auth/model"
)

// UserRepo returns brand repo
type UserRepo interface {
	GetUserById(ctx context.Context, id string) (*model.UserInfo, error)
	GetUserByPhoneOREmail(ctx context.Context, phone, email string) (*model.UserInfo, error)
	CreateUser(ctx context.Context, user *model.UserInfo) (*model.UserInfo, error)
}

// User brand repo
type User struct {
	table       *config.Table
	db          infra.DB
	vaultConfig *config.Vault
}

// NewUser returns new brand repo
func NewUser(table *config.Table, vaultConfig *config.Vault, db infra.DB) UserRepo {
	return &User{
		table:       table,
		vaultConfig: vaultConfig,
		db:          db,
	}
}

// GetUserByPhoneOREmail ....
func (p *User) GetUserByPhoneOREmail(ctx context.Context, phone, email string) (*model.UserInfo, error) {

	user := &model.UserInfo{}
	if err := p.db.FindOne(ctx, p.table.UserTable, infra.DbQuery{
		"phone": phone,
	}, []string{"UserOrgMap"}, user); err != nil {
		if err == infra.ErrNotFound {
			if err = p.db.FindOne(ctx, p.table.UserTable, infra.DbQuery{
				"email": email,
			}, []string{"UserOrgMap"}, user); err != nil {
				return nil, err
			}
		}
	}
	return user, nil
}

// GetUserById ....
func (p *User) GetUserById(ctx context.Context, id string) (*model.UserInfo, error) {
	user := &model.UserInfo{}
	if err := p.db.FindOne(ctx, p.table.UserTable, infra.DbQuery{
		"id": id,
	}, []string{"UserOrgMap"}, user); err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUser ....
func (p *User) CreateUser(ctx context.Context, user *model.UserInfo) (*model.UserInfo, error) {
	err := p.db.Insert(ctx, p.table.UserTable, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
