package repo

import (
	"context"
	"fmt"
	"github.com/triapex/auth/config"
	"github.com/triapex/auth/internal/infra"
	"github.com/triapex/auth/model"
	"go.mongodb.org/mongo-driver/bson"
	"io/ioutil"
	"path/filepath"
)

// UserRepo returns brand repo
type UserRepo interface {
	GetUserById(ctx context.Context, id string) (*model.UserInfo, error)
	GetUserByPhoneOREmail(ctx context.Context, phone, email string) (*model.UserInfo, error)
	CreateUser(ctx context.Context, user *model.UserInfo) (*model.UserInfo, error)
	CreateProfile(ctx context.Context, usrProfile *model.Profile) error
	SendResetPasswordCode(ctx context.Context, verification *model.VerificationInfo) error
	VerifyResetPasswordCode(ctx context.Context, id string) (*model.VerificationInfo, error)
	VerifyOtp(ctx context.Context, id string) (*model.VerificationInfo, error)
	ResetPassword(ctx context.Context, password string, id string) error
	SendOtp(ctx context.Context, verification *model.VerificationInfo) error
	UpdateUserVerificationStatus(ctx context.Context, id string, verified bool) error
	UpdateUserType(ctx context.Context, id uint, userType string) error
	StoreUserMSP(ctx context.Context, userName, mspPath string) error
	RemoveUserMSP(ctx context.Context, userName string) error
}

// User brand repo
type User struct {
	table       *config.Table
	db          infra.DB
	vault       infra.Vault
	vaultConfig *config.Vault
}

// NewUser returns new brand repo
func NewUser(table *config.Table, vaultConfig *config.Vault, db infra.DB, vault infra.Vault) UserRepo {
	return &User{
		table:       table,
		vaultConfig: vaultConfig,
		db:          db,
		vault:       vault,
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

// CreateProfile ...
func (p *User) CreateProfile(ctx context.Context, usrProfile *model.Profile) error {
	return p.db.Insert(ctx, p.table.UserCollectionNameProfile, usrProfile)
}

// SendResetPasswordCode ...
func (p *User) SendResetPasswordCode(ctx context.Context, verification *model.VerificationInfo) error {
	verificationInfo := &model.VerificationInfo{}
	err := p.db.FindOne(ctx, p.table.VerificationCollection, infra.DbQuery{
		"id": verification.ID,
	}, []string{"UserOrgMap"}, verificationInfo)
	if err != nil || verificationInfo == nil {
		return p.db.Insert(ctx, p.table.VerificationCollection, verification)
	}
	update := bson.D{{"$set",
		bson.D{
			{"code", verification.Code},
		},
	}}
	return p.db.UpdateOne(ctx, p.table.VerificationCollection, verificationInfo, update)
}

// VerifyResetPasswordCode - Does things done
func (p *User) VerifyResetPasswordCode(ctx context.Context, id string) (*model.VerificationInfo, error) {
	verification := &model.VerificationInfo{}
	if err := p.db.FindOne(ctx, p.table.VerificationCollection, infra.DbQuery{
		"id": id,
	}, []string{"UserOrgMap"}, verification); err != nil {
		return nil, err
	}
	return verification, nil
}

// ResetPassword ...
func (p *User) ResetPassword(ctx context.Context, password string, id string) error {
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set",
		bson.D{
			{"password", password},
		},
	}}
	if err := p.db.UpdateOne(ctx, p.table.UserTable, filter, update); err != nil {
		return err
	}
	return nil
}

func (p *User) SendOtp(ctx context.Context, verification *model.VerificationInfo) error {
	verificationInfo := &model.VerificationInfo{}
	err := p.db.FindOne(ctx, p.table.OtpCollection, infra.DbQuery{
		"id": verification.ID,
	}, []string{"UserOrgMap"}, verificationInfo)
	if err != nil || verificationInfo == nil {
		return p.db.Insert(ctx, p.table.OtpCollection, verification)
	}
	return p.db.UpdateOne(ctx, p.table.OtpCollection, infra.DbQuery{
		"id": verification.ID,
	}, verificationInfo)
}

func (p *User) VerifyOtp(ctx context.Context, id string) (*model.VerificationInfo, error) {
	verification := &model.VerificationInfo{}
	if err := p.db.FindOne(ctx, p.table.OtpCollection, infra.DbQuery{
		"id": id,
	}, []string{"UserOrgMap"}, verification); err != nil {
		return nil, err
	}
	return verification, nil
}

func (p *User) UpdateUserVerificationStatus(ctx context.Context, id string, verified bool) error {
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set",
		bson.D{
			{"verified", verified},
		},
	}}
	return p.db.UpdateOne(ctx, p.table.UserTable, filter, update)
}

func (p *User) UpdateUserType(ctx context.Context, id uint, userType string) error {
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set",
		bson.D{
			{"type", userType},
		},
	}}
	return p.db.UpdateOne(ctx, p.table.UserTable, filter, update)
}

func (p *User) StoreUserMSP(ctx context.Context, username, mspPath string) error {
	// Store IssuerRevocationPublicKey if exists
	revocationKeyPath := filepath.Join(mspPath, "IssuerRevocationPublicKey")
	if content, err := ioutil.ReadFile(revocationKeyPath); err == nil {
		vaultPath := fmt.Sprintf("%s/data/users/%s/IssuerRevocationPublicKey", p.vaultConfig.KvPath, username)
		if err := p.vault.LogicalWrite(vaultPath, content); err != nil {
			return fmt.Errorf("failed to store revocation key: %v", err)
		}

	}
	// Store signcerts
	signcertsDir := filepath.Join(mspPath, "signcerts")
	if err := p.vault.StoreDirectory(username, signcertsDir); err != nil {
		return fmt.Errorf("failed to store signcerts: %v", err)
	}

	// Store keystore
	keystoreDir := filepath.Join(mspPath, "keystore")
	if err := p.vault.StoreDirectory(username, keystoreDir); err != nil {
		return fmt.Errorf("failed to store keystore: %v", err)
	}

	// Store cacerts
	cacertsDir := filepath.Join(mspPath, "cacerts")
	if err := p.vault.StoreDirectory(username, cacertsDir); err != nil {
		return fmt.Errorf("failed to store cacerts: %v", err)
	}

	return nil
}

func (p *User) RemoveUserMSP(ctx context.Context, username string) error {
	err := p.vault.LogicalDelete(username)
	if err != nil {
		return err
	}
	return nil
}
