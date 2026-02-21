package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kaoyan/server/internal/middleware"
	"github.com/kaoyan/server/internal/pkg/response"
	"github.com/kaoyan/server/internal/service"
)

type PlanHandler struct {
	plannerService *service.PlannerService
}

func NewPlanHandler(plannerService *service.PlannerService) *PlanHandler {
	return &PlanHandler{plannerService: plannerService}
}

// GeneratePlan 生成学习计划
// POST /api/v1/plans/generate
func (h *PlanHandler) GeneratePlan(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req service.GeneratePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.UserID = userID

	plan, err := h.plannerService.GeneratePlan(&req)
	if err != nil {
		response.ServerError(c, "生成计划失败: "+err.Error())
		return
	}

	response.OK(c, plan)
}

// GetPlan 获取计划详情
// GET /api/v1/plans/:id
func (h *PlanHandler) GetPlan(c *gin.Context) {
	planID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的计划ID")
		return
	}

	plan, err := h.plannerService.GetPlan(planID)
	if err != nil {
		response.NotFound(c, "计划不存在")
		return
	}

	// IDOR 防护：验证计划归属
	userID := middleware.GetUserID(c)
	if plan.UserID != userID {
		response.NotFound(c, "计划不存在")
		return
	}

	response.OK(c, plan)
}

// ListPlans 获取计划列表
// GET /api/v1/plans
func (h *PlanHandler) ListPlans(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	plans, total, err := h.plannerService.ListPlans(userID, page, pageSize)
	if err != nil {
		response.ServerError(c, "获取计划列表失败")
		return
	}

	response.OKWithPage(c, plans, total, page, pageSize)
}

// GetTodayTasks 获取今日任务
// GET /api/v1/plans/today
func (h *PlanHandler) GetTodayTasks(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tasks, err := h.plannerService.GetTodayTasks(userID)
	if err != nil {
		response.ServerError(c, "获取今日任务失败")
		return
	}
	response.OK(c, tasks)
}

// GetWeekTasks 获取本周任务
// GET /api/v1/plans/week
func (h *PlanHandler) GetWeekTasks(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tasks, err := h.plannerService.GetWeekTasks(userID)
	if err != nil {
		response.ServerError(c, "获取本周任务失败")
		return
	}
	response.OK(c, tasks)
}

// UpdateTaskStatus 更新任务状态
// PATCH /api/v1/tasks/:id/status
func (h *PlanHandler) UpdateTaskStatus(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的任务ID")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=pending in_progress completed skipped"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: status 必须是 pending/in_progress/completed/skipped")
		return
	}

	// IDOR 防护：将 userID 传入 service 层校验任务归属
	userID := middleware.GetUserID(c)
	if err := h.plannerService.UpdateTaskStatus(taskID, req.Status, userID); err != nil {
		response.ServerError(c, "更新状态失败")
		return
	}

	response.OKWithMessage(c, "更新成功")
}

// RefreshPlan 手动刷新计划
// POST /api/v1/plans/:id/refresh
func (h *PlanHandler) RefreshPlan(c *gin.Context) {
	planID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的计划ID")
		return
	}

	// IDOR 防护：验证计划归属
	userID := middleware.GetUserID(c)
	existingPlan, err := h.plannerService.GetPlan(planID)
	if err != nil || existingPlan.UserID != userID {
		response.NotFound(c, "计划不存在")
		return
	}

	plan, err := h.plannerService.RefreshPlan(planID)
	if err != nil {
		response.ServerError(c, "刷新计划失败: "+err.Error())
		return
	}

	response.OK(c, plan)
}
