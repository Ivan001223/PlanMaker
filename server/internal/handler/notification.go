package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kaoyan/server/internal/middleware"
	"github.com/kaoyan/server/internal/pkg/response"
	"github.com/kaoyan/server/internal/service"
)

type NotificationHandler struct {
	notifService *service.NotificationService
}

func NewNotificationHandler(notifService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

// ListNotifications 获取通知列表
// GET /api/v1/notifications
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	notifications, total, err := h.notifService.ListByUser(userID, page, pageSize)
	if err != nil {
		response.ServerError(c, "获取通知失败")
		return
	}

	response.OKWithPage(c, notifications, total, page, pageSize)
}

// GenerateNotifications 为计划生成通知
// POST /api/v1/notifications/generate
func (h *NotificationHandler) GenerateNotifications(c *gin.Context) {
	var req struct {
		PlanID uint64 `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// IDOR 防护：将 userID 传入 service 层校验计划归属
	userID := middleware.GetUserID(c)
	if err := h.notifService.GenerateTaskNotifications(req.PlanID, userID); err != nil {
		response.ServerError(c, "生成通知失败: "+err.Error())
		return
	}

	response.OKWithMessage(c, "通知已生成")
}
