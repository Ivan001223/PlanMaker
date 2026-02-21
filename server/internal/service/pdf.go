package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/kaoyan/server/internal/model"
)

type PDFService struct {
	db            *gorm.DB
	pdfServiceURL string
	logger        *zap.Logger
}

func NewPDFService(db *gorm.DB, pdfServiceURL string, logger *zap.Logger) *PDFService {
	return &PDFService{
		db:            db,
		pdfServiceURL: pdfServiceURL,
		logger:        logger,
	}
}

// CreateTextbook 创建教材记录
func (s *PDFService) CreateTextbook(userID uint64, title, subject, fileKey string, fileSize int64) (*model.Textbook, error) {
	textbook := &model.Textbook{
		UserID:      userID,
		Title:       title,
		Subject:     subject,
		FileKey:     fileKey,
		FileSize:    fileSize,
		ParseStatus: "pending",
	}
	if err := s.db.Create(textbook).Error; err != nil {
		return nil, err
	}
	return textbook, nil
}

// TriggerParse 触发PDF解析
func (s *PDFService) TriggerParse(textbookID uint64) error {
	textbook := &model.Textbook{}
	if err := s.db.First(textbook, textbookID).Error; err != nil {
		return err
	}

	payload := map[string]interface{}{
		"textbook_id": textbook.ID,
		"user_id":     textbook.UserID,
		"file_key":    textbook.FileKey,
		"subject":     textbook.Subject,
		"title":       textbook.Title,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 调用 PDF 解析服务
	url := fmt.Sprintf("%s/api/v1/parse", s.pdfServiceURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Warn("调用PDF解析服务失败", zap.Error(err))
		// 不阻塞主流程，标记为 processing
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			s.logger.Warn("PDF解析服务返回错误", zap.Int("status", resp.StatusCode))
		}
	}

	textbook.ParseStatus = "processing"
	return s.db.Save(textbook).Error
}

// GetTextbook 获取教材信息
func (s *PDFService) GetTextbook(id uint64) (*model.Textbook, error) {
	var textbook model.Textbook
	err := s.db.First(&textbook, id).Error
	return &textbook, err
}

// ListTextbooks 获取用户的教材列表
func (s *PDFService) ListTextbooks(userID uint64) ([]model.Textbook, error) {
	var textbooks []model.Textbook
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&textbooks).Error
	return textbooks, err
}

// DeleteTextbook 删除教材
func (s *PDFService) DeleteTextbook(id uint64) error {
	return s.db.Delete(&model.Textbook{}, id).Error
}

// UpdateParseResult 更新解析结果 (由PDF服务回调)
func (s *PDFService) UpdateParseResult(textbookID uint64, mongoDocID string, chapters, knowledgePoints int) error {
	return s.db.Model(&model.Textbook{}).Where("id = ?", textbookID).Updates(map[string]interface{}{
		"parse_status":           "completed",
		"mongo_doc_id":           mongoDocID,
		"total_chapters":         chapters,
		"total_knowledge_points": knowledgePoints,
		"updated_at":             time.Now(),
	}).Error
}
