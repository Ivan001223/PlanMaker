package repository

import (
	"github.com/kaoyan/server/internal/model"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// FindByOpenID 根据 OpenID 查找用户
func (r *UserRepo) FindByOpenID(openID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("openid = ?", openID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 根据 ID 查找用户
func (r *UserRepo) FindByID(id uint64) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Create 创建用户
func (r *UserRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// Update 更新用户
func (r *UserRepo) Update(user *model.User) error {
	return r.db.Save(user).Error
}

// GetPreference 获取用户偏好
func (r *UserRepo) GetPreference(userID uint64) (*model.UserPreference, error) {
	var pref model.UserPreference
	err := r.db.Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

// SavePreference 保存用户偏好 (upsert)
func (r *UserRepo) SavePreference(pref *model.UserPreference) error {
	var existing model.UserPreference
	err := r.db.Where("user_id = ?", pref.UserID).First(&existing).Error
	if err == nil {
		// 已存在，设置 ID 以便 GORM 执行 UPDATE 而非 INSERT
		pref.ID = existing.ID
	}
	return r.db.Save(pref).Error
}
