-- =============================================
-- 考研规划小程序 - MySQL 数据库初始化
-- =============================================

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- -------------------------------------------
-- 1. 用户表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    `openid` VARCHAR(64) NOT NULL COMMENT '微信OpenID',
    `union_id` VARCHAR(64) DEFAULT NULL COMMENT '微信UnionID',
    `nickname` VARCHAR(64) DEFAULT '' COMMENT '昵称',
    `avatar_url` VARCHAR(512) DEFAULT '' COMMENT '头像URL',
    `phone` VARCHAR(20) DEFAULT NULL COMMENT '手机号',
    `membership` ENUM('free', 'pro', 'sprint') DEFAULT 'free' COMMENT '会员等级',
    `membership_expire_at` DATETIME DEFAULT NULL COMMENT '会员过期时间',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_openid` (`openid`),
    KEY `idx_union_id` (`union_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- -------------------------------------------
-- 2. 用户偏好设置表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS `user_preferences` (
    `id` BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `exam_date` DATE DEFAULT NULL COMMENT '考试日期',
    `target_score` INT DEFAULT NULL COMMENT '目标分数',
    `daily_study_hours` DECIMAL(3,1) DEFAULT 8.0 COMMENT '每日学习时长(小时)',
    `weak_subjects` JSON DEFAULT NULL COMMENT '薄弱科目 ["math","english"]',
    `study_start_time` TIME DEFAULT '08:00:00' COMMENT '每日学习开始时间',
    `study_end_time` TIME DEFAULT '22:00:00' COMMENT '每日学习结束时间',
    `rest_days` JSON DEFAULT NULL COMMENT '固定休息日 [0,6] (周日,周六)',
    `pomodoro_duration` INT DEFAULT 25 COMMENT '番茄钟时长(分钟)',
    `break_duration` INT DEFAULT 5 COMMENT '短休息时长(分钟)',
    `long_break_duration` INT DEFAULT 15 COMMENT '长休息时长(分钟)',
    `long_break_interval` INT DEFAULT 4 COMMENT '长休息间隔(几个番茄钟)',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_user_id` (`user_id`),
    CONSTRAINT `fk_pref_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户偏好设置';

-- -------------------------------------------
-- 3. 教材表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS `textbooks` (
    `id` BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `title` VARCHAR(256) NOT NULL COMMENT '教材名称',
    `subject` ENUM('math', 'english', 'politics', 'professional') NOT NULL COMMENT '科目',
    `file_key` VARCHAR(512) DEFAULT NULL COMMENT 'MinIO文件key',
    `file_size` BIGINT DEFAULT 0 COMMENT '文件大小(bytes)',
    `parse_status` ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending' COMMENT '解析状态',
    `parse_task_id` VARCHAR(128) DEFAULT NULL COMMENT '解析任务ID',
    `total_chapters` INT DEFAULT 0 COMMENT '总章节数',
    `total_knowledge_points` INT DEFAULT 0 COMMENT '总知识点数',
    `mongo_doc_id` VARCHAR(64) DEFAULT NULL COMMENT 'MongoDB解析结果文档ID',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY `idx_user_id` (`user_id`),
    KEY `idx_parse_status` (`parse_status`),
    CONSTRAINT `fk_textbook_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='教材表';

-- -------------------------------------------
-- 4. 学习计划主表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS `study_plans` (
    `id` BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `plan_name` VARCHAR(128) NOT NULL COMMENT '计划名称',
    `exam_date` DATE NOT NULL COMMENT '考试日期',
    `start_date` DATE NOT NULL COMMENT '计划开始日期',
    `status` ENUM('active', 'paused', 'completed', 'abandoned') DEFAULT 'active' COMMENT '状态',
    `plan_type` ENUM('full', 'weekly', 'daily') DEFAULT 'full' COMMENT '计划类型',
    `version` INT DEFAULT 1 COMMENT '版本号',
    `total_tasks` INT DEFAULT 0 COMMENT '总任务数',
    `completed_tasks` INT DEFAULT 0 COMMENT '已完成任务数',
    `ai_prompt` TEXT DEFAULT NULL COMMENT 'AI生成时的prompt',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY `idx_user_id` (`user_id`),
    KEY `idx_status` (`status`),
    CONSTRAINT `fk_plan_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='学习计划主表';

-- -------------------------------------------
-- 5. 学习任务表 (按日期分区)
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS `study_tasks` (
    `id` BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    `plan_id` BIGINT UNSIGNED NOT NULL COMMENT '计划ID',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `task_date` DATE NOT NULL COMMENT '任务日期',
    `start_time` TIME NOT NULL COMMENT '开始时间',
    `end_time` TIME NOT NULL COMMENT '结束时间',
    `content` TEXT NOT NULL COMMENT '任务内容',
    `task_type` ENUM('study', 'review', 'break', 'mock_exam') DEFAULT 'study' COMMENT '任务类型',
    `subject` ENUM('math', 'english', 'politics', 'professional') DEFAULT NULL COMMENT '科目',
    `chapter` VARCHAR(128) DEFAULT NULL COMMENT '章节',
    `knowledge_points` JSON DEFAULT NULL COMMENT '关联知识点',
    `pomodoro_count` INT DEFAULT 1 COMMENT '番茄钟数',
    `status` ENUM('pending', 'in_progress', 'completed', 'skipped') DEFAULT 'pending' COMMENT '状态',
    `actual_start_time` DATETIME DEFAULT NULL COMMENT '实际开始时间',
    `actual_end_time` DATETIME DEFAULT NULL COMMENT '实际结束时间',
    `notes` TEXT DEFAULT NULL COMMENT '笔记',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY `idx_plan_id` (`plan_id`),
    KEY `idx_user_date` (`user_id`, `task_date`),
    KEY `idx_task_date` (`task_date`),
    KEY `idx_status` (`status`),
    CONSTRAINT `fk_task_plan` FOREIGN KEY (`plan_id`) REFERENCES `study_plans`(`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_task_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='学习任务表';

-- -------------------------------------------
-- 6. 对话会话表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS `chat_sessions` (
    `id` BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `session_type` ENUM('planning', 'adjustment', 'qa') DEFAULT 'planning' COMMENT '会话类型',
    `title` VARCHAR(128) DEFAULT '' COMMENT '会话标题',
    `status` ENUM('active', 'closed') DEFAULT 'active' COMMENT '状态',
    `context` JSON DEFAULT NULL COMMENT '对话上下文(收集到的信息)',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY `idx_user_id` (`user_id`),
    CONSTRAINT `fk_session_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话会话表';

-- -------------------------------------------
-- 7. 对话消息表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS `chat_messages` (
    `id` BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    `session_id` BIGINT UNSIGNED NOT NULL COMMENT '会话ID',
    `role` ENUM('user', 'assistant', 'system') NOT NULL COMMENT '角色',
    `content` TEXT NOT NULL COMMENT '消息内容',
    `tokens_used` INT DEFAULT 0 COMMENT '消耗token数',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_session_id` (`session_id`),
    CONSTRAINT `fk_msg_session` FOREIGN KEY (`session_id`) REFERENCES `chat_sessions`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话消息表';

-- -------------------------------------------
-- 8. 通知记录表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS `notifications` (
    `id` BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `notify_type` ENUM('study_start', 'break', 'review', 'daily_summary') NOT NULL COMMENT '通知类型',
    `title` VARCHAR(128) NOT NULL COMMENT '通知标题',
    `content` VARCHAR(512) NOT NULL COMMENT '通知内容',
    `scheduled_at` DATETIME NOT NULL COMMENT '计划发送时间',
    `sent_at` DATETIME DEFAULT NULL COMMENT '实际发送时间',
    `status` ENUM('pending', 'sent', 'failed', 'cancelled') DEFAULT 'pending' COMMENT '状态',
    `related_task_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联任务ID',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_user_id` (`user_id`),
    KEY `idx_scheduled_at` (`scheduled_at`),
    KEY `idx_status` (`status`),
    CONSTRAINT `fk_notif_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知记录表';
