package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kaoyan/server/internal/middleware"
	"github.com/kaoyan/server/internal/pkg/response"
	"github.com/kaoyan/server/internal/service"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// CreateSession 创建对话会话
// POST /api/v1/chat/sessions
func (h *ChatHandler) CreateSession(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		SessionType string `json:"session_type" binding:"omitempty,oneof=planning adjustment qa"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.SessionType = "planning"
	}
	if req.SessionType == "" {
		req.SessionType = "planning"
	}

	session, err := h.chatService.CreateSession(userID, req.SessionType)
	if err != nil {
		response.ServerError(c, "创建会话失败")
		return
	}
	response.OK(c, session)
}

// GetSession 获取会话详情（含消息历史）
// GET /api/v1/chat/sessions/:id
func (h *ChatHandler) GetSession(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	session, err := h.chatService.GetSession(sessionID)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	// IDOR 防护：验证会话归属
	userID := middleware.GetUserID(c)
	if session.UserID != userID {
		response.NotFound(c, "会话不存在")
		return
	}

	response.OK(c, session)
}

// ListSessions 获取会话列表
// GET /api/v1/chat/sessions
func (h *ChatHandler) ListSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	sessions, err := h.chatService.ListSessions(userID)
	if err != nil {
		response.ServerError(c, "获取会话列表失败")
		return
	}
	response.OK(c, sessions)
}

// SendMessage 发送消息
// POST /api/v1/chat/sessions/:id/messages
func (h *ChatHandler) SendMessage(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	// IDOR 防护：验证会话归属
	userID := middleware.GetUserID(c)
	session, err := h.chatService.GetSession(sessionID)
	if err != nil || session.UserID != userID {
		response.NotFound(c, "会话不存在")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "消息内容不能为空")
		return
	}

	const maxMessageLen = 5000
	if len([]rune(req.Content)) > maxMessageLen {
		response.BadRequest(c, "消息内容超过最大长度限制")
		return
	}

	msg, err := h.chatService.SendMessage(sessionID, req.Content)
	if err != nil {
		response.ServerError(c, "发送消息失败: "+err.Error())
		return
	}
	response.OK(c, msg)
}
