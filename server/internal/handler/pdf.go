package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kaoyan/server/internal/middleware"
	"github.com/kaoyan/server/internal/pkg/response"
	"github.com/kaoyan/server/internal/service"
)

type PDFHandler struct {
	pdfService *service.PDFService
}

func NewPDFHandler(pdfService *service.PDFService) *PDFHandler {
	return &PDFHandler{pdfService: pdfService}
}

// UploadPDF 上传PDF（接受元数据，实际文件上传到MinIO）
// POST /api/v1/textbooks
func (h *PDFHandler) UploadPDF(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Title    string `json:"title" binding:"required"`
		Subject  string `json:"subject" binding:"required,oneof=math english politics professional"`
		FileKey  string `json:"file_key" binding:"required"`
		FileSize int64  `json:"file_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	textbook, err := h.pdfService.CreateTextbook(userID, req.Title, req.Subject, req.FileKey, req.FileSize)
	if err != nil {
		response.ServerError(c, "创建教材记录失败")
		return
	}

	// 触发异步解析
	go func() {
		_ = h.pdfService.TriggerParse(textbook.ID)
	}()

	response.OK(c, textbook)
}

// ListTextbooks 获取教材列表
// GET /api/v1/textbooks
func (h *PDFHandler) ListTextbooks(c *gin.Context) {
	userID := middleware.GetUserID(c)
	textbooks, err := h.pdfService.ListTextbooks(userID)
	if err != nil {
		response.ServerError(c, "获取教材列表失败")
		return
	}
	response.OK(c, textbooks)
}

// ParseCallback PDF解析回调
// POST /api/v1/textbooks/parse-callback (内部调用)
func (h *PDFHandler) ParseCallback(c *gin.Context) {
	var req struct {
		TextbookID      uint64 `json:"textbook_id" binding:"required"`
		MongoDocID      string `json:"mongo_doc_id" binding:"required"`
		TotalChapters   int    `json:"total_chapters"`
		KnowledgePoints int    `json:"knowledge_points"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.pdfService.UpdateParseResult(req.TextbookID, req.MongoDocID, req.TotalChapters, req.KnowledgePoints); err != nil {
		response.ServerError(c, "更新解析结果失败")
		return
	}

	response.OKWithMessage(c, "更新成功")
}
