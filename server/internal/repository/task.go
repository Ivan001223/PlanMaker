package repository

import (
	"github.com/kaoyan/server/internal/model"
	"gorm.io/gorm"
)

type TaskRepo struct {
	db *gorm.DB
}

func NewTaskRepo(db *gorm.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// DB 返回数据库实例
func (r *TaskRepo) DB() *gorm.DB {
	return r.db
}

// FindByIDTx 使用指定事务查找任务
func (r *TaskRepo) FindByIDTx(tx *gorm.DB, id uint64) (*model.StudyTask, error) {
	var task model.StudyTask
	err := tx.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// BatchCreate 批量创建任务
func (r *TaskRepo) BatchCreate(tasks []model.StudyTask) error {
	return r.db.CreateInBatches(tasks, 100).Error
}

// FindByID 根据ID查找任务
func (r *TaskRepo) FindByID(id uint64) (*model.StudyTask, error) {
	var task model.StudyTask
	err := r.db.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListByDate 查询某天的任务
func (r *TaskRepo) ListByDate(userID uint64, date string) ([]model.StudyTask, error) {
	var tasks []model.StudyTask
	err := r.db.Where("user_id = ? AND task_date = ?", userID, date).
		Order("start_time ASC").
		Find(&tasks).Error
	return tasks, err
}

// ListByDateRange 查询日期范围内的任务
func (r *TaskRepo) ListByDateRange(userID uint64, startDate, endDate string) ([]model.StudyTask, error) {
	var tasks []model.StudyTask
	err := r.db.Where("user_id = ? AND task_date BETWEEN ? AND ?", userID, startDate, endDate).
		Order("task_date ASC, start_time ASC").
		Find(&tasks).Error
	return tasks, err
}

// UpdateStatus 更新任务状态
func (r *TaskRepo) UpdateStatus(id uint64, status string) error {
	return r.db.Model(&model.StudyTask{}).Where("id = ?", id).
		Update("status", status).Error
}

// Update 更新任务
func (r *TaskRepo) Update(task *model.StudyTask) error {
	return r.db.Save(task).Error
}

// DeleteByPlanID 删除计划下的所有任务
func (r *TaskRepo) DeleteByPlanID(planID uint64) error {
	return r.db.Where("plan_id = ?", planID).Delete(&model.StudyTask{}).Error
}

// GetCompletionStats 获取完成统计
func (r *TaskRepo) GetCompletionStats(planID uint64) (total int64, completed int64, err error) {
	r.db.Model(&model.StudyTask{}).Where("plan_id = ?", planID).Count(&total)
	r.db.Model(&model.StudyTask{}).Where("plan_id = ? AND status = 'completed'", planID).Count(&completed)
	return
}

// ListPendingBefore 查询某日期之前未完成的任务（过期未完成）
func (r *TaskRepo) ListPendingBefore(userID, planID uint64, beforeDate string) ([]model.StudyTask, error) {
	var tasks []model.StudyTask
	err := r.db.Where("user_id = ? AND plan_id = ? AND task_date < ? AND status = 'pending'",
		userID, planID, beforeDate).
		Order("task_date ASC, start_time ASC").
		Find(&tasks).Error
	return tasks, err
}

// DeleteFuturePending 删除未来的 pending 状态任务（用于刷新计划）
func (r *TaskRepo) DeleteFuturePending(planID uint64, fromDate string) error {
	return r.db.Where("plan_id = ? AND task_date >= ? AND status = 'pending'",
		planID, fromDate).Delete(&model.StudyTask{}).Error
}
