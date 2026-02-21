package service

import (
	"errors"
	"log"

	"github.com/kaoyan/server/internal/middleware"
	"github.com/kaoyan/server/internal/model"
	"github.com/kaoyan/server/internal/pkg/wechat"
	"github.com/kaoyan/server/internal/repository"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo     *repository.UserRepo
	wechatClient *wechat.Client
	jwtSecret    string
	jwtExpire    int
}

func NewUserService(userRepo *repository.UserRepo, wechatClient *wechat.Client, jwtSecret string, jwtExpire int) *UserService {
	return &UserService{
		userRepo:     userRepo,
		wechatClient: wechatClient,
		jwtSecret:    jwtSecret,
		jwtExpire:    jwtExpire,
	}
}

// LoginResult 登录结果
type LoginResult struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
	IsNew bool        `json:"is_new"`
}

// WeChatLogin 微信登录
func (s *UserService) WeChatLogin(code string) (*LoginResult, error) {
	// 1. 调用微信 code2session
	sessionResp, err := s.wechatClient.Code2Session(code)
	if err != nil {
		return nil, err
	}

	// 2. 查找或创建用户
	user, err := s.userRepo.FindByOpenID(sessionResp.OpenID)
	isNew := false
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 新用户
			user = &model.User{
				OpenID:  sessionResp.OpenID,
				UnionID: sessionResp.UnionID,
			}
			if err := s.userRepo.Create(user); err != nil {
				return nil, err
			}
			// 创建默认偏好
			pref := &model.UserPreference{
				UserID: user.ID,
			}
			if err := s.userRepo.SavePreference(pref); err != nil {
				log.Printf("创建默认偏好失败: %v", err)
			}
			isNew = true
		} else {
			return nil, err
		}
	}

	// 3. 生成 JWT
	token, err := middleware.GenerateToken(user.ID, user.OpenID, s.jwtExpire)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token: token,
		User:  user,
		IsNew: isNew,
	}, nil
}

// GetProfile 获取用户资料
func (s *UserService) GetProfile(userID uint64) (*model.User, error) {
	return s.userRepo.FindByID(userID)
}

// UpdateProfile 更新用户资料
func (s *UserService) UpdateProfile(userID uint64, nickname, avatarURL string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	if nickname != "" {
		user.Nickname = nickname
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}
	return s.userRepo.Update(user)
}

// GetPreference 获取用户偏好
func (s *UserService) GetPreference(userID uint64) (*model.UserPreference, error) {
	pref, err := s.userRepo.GetPreference(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建默认偏好
			pref = &model.UserPreference{UserID: userID}
			if err := s.userRepo.SavePreference(pref); err != nil {
				return nil, err
			}
			return pref, nil
		}
		return nil, err
	}
	return pref, nil
}

// UpdatePreference 更新用户偏好
func (s *UserService) UpdatePreference(userID uint64, pref *model.UserPreference) error {
	pref.UserID = userID
	return s.userRepo.SavePreference(pref)
}
