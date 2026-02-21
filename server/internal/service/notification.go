package service

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/kaoyan/server/internal/model"
	"github.com/kaoyan/server/internal/repository"
)

type NotificationService struct {
	notifRepo *repository.NotificationRepo
	logger    *zap.Logger
}

func NewNotificationService(notifRepo *repository.NotificationRepo, logger *zap.Logger) *NotificationService {
	return &NotificationService{
		notifRepo: notifRepo,
		logger:    logger,
	}
}

func (s *NotificationService) CreateNotification(userID uint64, notifyType, title, content string, scheduledAt time.Time, relatedTaskID *uint64) (*model.Notification, error) {
	notification := &model.Notification{
		UserID:        userID,
		NotifyType:    notifyType,
		Title:         title,
		Content:       content,
		ScheduledAt:   scheduledAt,
		Status:        "pending",
		RelatedTaskID: relatedTaskID,
	}
	if err := s.notifRepo.Create(notification); err != nil {
		return nil, err
	}
	return notification, nil
}

func (s *NotificationService) GetPendingNotifications() ([]model.Notification, error) {
	return s.notifRepo.FindPending(100)
}

func (s *NotificationService) MarkSent(id uint64) error {
	now := time.Now()
	return s.notifRepo.UpdateStatus(id, "sent", &now)
}

func (s *NotificationService) MarkFailed(id uint64) error {
	return s.notifRepo.UpdateStatus(id, "failed", nil)
}

func (s *NotificationService) ListByUser(userID uint64, page, pageSize int) ([]model.Notification, int64, error) {
	return s.notifRepo.ListByUserID(userID, page, pageSize)
}

func (s *NotificationService) GenerateTaskNotifications(planID uint64, userID uint64) error {
	plan, err := s.notifRepo.FindPlanByID(planID)
	if err != nil {
		return fmt.Errorf("计划不存在")
	}
	if plan.UserID != userID {
		return fmt.Errorf("无权操作此计划")
	}

	tasks, err := s.notifRepo.FindPendingTasksByPlanID(planID)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		taskDateTime, err := time.Parse("2006-01-02 15:04", task.TaskDate+" "+task.StartTime)
		if err != nil {
			s.logger.Warn("解析任务时间失败", zap.Error(err), zap.Uint64("task_id", task.ID))
			continue
		}

		scheduledAt := taskDateTime.Add(-5 * time.Minute)
		if scheduledAt.Before(time.Now()) {
			continue
		}

		taskID := task.ID
		_, err = s.CreateNotification(
			task.UserID,
			"study_start",
			"学习提醒",
			subjectName(task.Subject)+" 学习即将开始："+task.Content,
			scheduledAt,
			&taskID,
		)
		if err != nil {
			s.logger.Error("创建通知失败", zap.Error(err))
		}
	}

	return nil
}
