package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kaoyan/server/internal/middleware"
	"github.com/kaoyan/server/internal/model"
	"github.com/kaoyan/server/internal/pkg/response"
	"github.com/kaoyan/server/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Login 微信登录
// POST /api/v1/auth/login
func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少登录code")
		return
	}

	result, err := h.userService.WeChatLogin(req.Code)
	if err != nil {
		response.ServerError(c, "登录失败: "+err.Error())
		return
	}

	response.OK(c, result)
}

// GetProfile 获取个人资料
// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.userService.GetProfile(userID)
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}
	response.OK(c, user)
}

// UpdateProfile 更新个人资料
// PUT /api/v1/user/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.userService.UpdateProfile(userID, req.Nickname, req.AvatarURL); err != nil {
		response.ServerError(c, "更新失败")
		return
	}
	response.OKWithMessage(c, "更新成功")
}

// GetPreference 获取偏好设置
// GET /api/v1/user/preference
func (h *UserHandler) GetPreference(c *gin.Context) {
	userID := middleware.GetUserID(c)
	pref, err := h.userService.GetPreference(userID)
	if err != nil {
		response.ServerError(c, "获取偏好失败")
		return
	}
	response.OK(c, pref)
}

// UpdatePreference 更新偏好设置
// PUT /api/v1/user/preference
func (h *UserHandler) UpdatePreference(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var pref model.UserPreference
	if err := c.ShouldBindJSON(&pref); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.userService.UpdatePreference(userID, &pref); err != nil {
		response.ServerError(c, "更新偏好失败")
		return
	}
	response.OKWithMessage(c, "更新成功")
}
