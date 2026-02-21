-- =============================================
-- 考研规划小程序 - PostgreSQL 数据库初始化
-- =============================================

-- -------------------------------------------
-- 1. 用户表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    openid VARCHAR(64) NOT NULL,
    union_id VARCHAR(64) DEFAULT NULL,
    nickname VARCHAR(64) DEFAULT '',
    avatar_url VARCHAR(512) DEFAULT '',
    phone VARCHAR(20) DEFAULT NULL,
    membership VARCHAR(20) DEFAULT 'free',
    membership_expire_at TIMESTAMPTZ DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_openid ON users(openid);
CREATE INDEX IF NOT EXISTS idx_union_id ON users(union_id);

-- -------------------------------------------
-- 2. 用户偏好设置表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS user_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exam_date DATE DEFAULT NULL,
    target_score INT DEFAULT NULL,
    daily_study_hours DECIMAL(3,1) DEFAULT 8.0,
    weak_subjects JSONB DEFAULT NULL,
    study_start_time TIME DEFAULT '08:00:00',
    study_end_time TIME DEFAULT '22:00:00',
    rest_days JSONB DEFAULT NULL,
    pomodoro_duration INT DEFAULT 25,
    break_duration INT DEFAULT 5,
    long_break_duration INT DEFAULT 15,
    long_break_interval INT DEFAULT 4,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_user_pref_user_id ON user_preferences(user_id);

-- -------------------------------------------
-- 3. 教材表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS textbooks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(256) NOT NULL,
    subject VARCHAR(20) NOT NULL,
    file_key VARCHAR(512) DEFAULT NULL,
    file_size BIGINT DEFAULT 0,
    parse_status VARCHAR(20) DEFAULT 'pending',
    parse_task_id VARCHAR(128) DEFAULT NULL,
    total_chapters INT DEFAULT 0,
    total_knowledge_points INT DEFAULT 0,
    mongo_doc_id VARCHAR(64) DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_textbook_user_id ON textbooks(user_id);
CREATE INDEX IF NOT EXISTS idx_parse_status ON textbooks(parse_status);

-- -------------------------------------------
-- 4. 学习计划主表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS study_plans (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_name VARCHAR(128) NOT NULL,
    exam_date DATE NOT NULL,
    start_date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    plan_type VARCHAR(20) DEFAULT 'full',
    version INT DEFAULT 1,
    total_tasks INT DEFAULT 0,
    completed_tasks INT DEFAULT 0,
    ai_prompt TEXT DEFAULT NULL,
    target_school VARCHAR(128) DEFAULT NULL,
    target_major VARCHAR(128) DEFAULT NULL,
    materials JSONB DEFAULT NULL,
    plan_phases JSONB DEFAULT NULL,
    last_refresh_at TIMESTAMPTZ DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_plan_user_id ON study_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_plan_status ON study_plans(status);

-- -------------------------------------------
-- 5. 学习任务表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS study_tasks (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES study_plans(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    content TEXT NOT NULL,
    task_type VARCHAR(20) DEFAULT 'study',
    subject VARCHAR(20) DEFAULT NULL,
    chapter VARCHAR(128) DEFAULT NULL,
    knowledge_points JSONB DEFAULT NULL,
    pomodoro_count INT DEFAULT 1,
    status VARCHAR(20) DEFAULT 'pending',
    actual_start_time TIMESTAMPTZ DEFAULT NULL,
    actual_end_time TIMESTAMPTZ DEFAULT NULL,
    notes TEXT DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_task_plan_id ON study_tasks(plan_id);
CREATE INDEX IF NOT EXISTS idx_task_user_date ON study_tasks(user_id, task_date);
CREATE INDEX IF NOT EXISTS idx_task_date ON study_tasks(task_date);
CREATE INDEX IF NOT EXISTS idx_task_status ON study_tasks(status);

-- -------------------------------------------
-- 6. 对话会话表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS chat_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_type VARCHAR(20) DEFAULT 'planning',
    title VARCHAR(128) DEFAULT '',
    status VARCHAR(20) DEFAULT 'active',
    context JSONB DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_session_user_id ON chat_sessions(user_id);

-- -------------------------------------------
-- 7. 对话消息表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    tokens_used INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_msg_session_id ON chat_messages(session_id);

-- -------------------------------------------
-- 8. 通知记录表
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notify_type VARCHAR(20) NOT NULL,
    title VARCHAR(128) NOT NULL,
    content VARCHAR(512) NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ DEFAULT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    related_task_id BIGINT DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_notif_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notif_scheduled_at ON notifications(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_notif_status ON notifications(status);
