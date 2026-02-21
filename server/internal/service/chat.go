package service

import (
	"fmt"

	"github.com/kaoyan/server/internal/model"
	"github.com/kaoyan/server/internal/pkg/llm"
	"github.com/kaoyan/server/internal/repository"
	"go.uber.org/zap"
)

type ChatService struct {
	chatRepo *repository.ChatRepo
	aiClient *llm.Client
	aiModel  string
	logger   *zap.Logger
}

func NewChatService(chatRepo *repository.ChatRepo, aiClient *llm.Client, aiModel string, logger *zap.Logger) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
		aiClient: aiClient,
		aiModel:  aiModel,
		logger:   logger,
	}
}

func (s *ChatService) CreateSession(userID uint64, sessionType string) (*model.ChatSession, error) {
	session := &model.ChatSession{
		UserID:      userID,
		SessionType: sessionType,
		Status:      "active",
	}
	if err := s.chatRepo.CreateSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *ChatService) GetSession(sessionID uint64) (*model.ChatSession, error) {
	return s.chatRepo.FindSessionByID(sessionID)
}

func (s *ChatService) ListSessions(userID uint64) ([]model.ChatSession, error) {
	return s.chatRepo.ListSessionsByUserID(userID)
}

func (s *ChatService) SendMessage(sessionID uint64, userMessage string) (*model.ChatMessage, error) {
	userMsg := &model.ChatMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   userMessage,
	}
	if err := s.chatRepo.CreateMessage(userMsg); err != nil {
		return nil, err
	}

	history, _ := s.chatRepo.ListRecentMessages(sessionID, 20)

	var messages []llm.Message
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: chatSystemPrompt,
	})
	for _, msg := range history {
		messages = append(messages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	aiResp, err := s.aiClient.ChatCompletion(llm.ChatRequest{
		Model:       s.aiModel,
		Messages:    messages,
		Temperature: 0.8,
		MaxTokens:   2048,
	})
	if err != nil {
		s.logger.Error("AI对话失败", zap.Error(err))
		return nil, fmt.Errorf("AI对话失败: %w", err)
	}

	if len(aiResp.Choices) == 0 {
		return nil, fmt.Errorf("AI返回空响应")
	}

	aiContent := aiResp.Choices[0].Message.Content
	tokensUsed := aiResp.Usage.TotalTokens

	assistantMsg := &model.ChatMessage{
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    aiContent,
		TokensUsed: tokensUsed,
	}
	if err := s.chatRepo.CreateMessage(assistantMsg); err != nil {
		return nil, err
	}

	return assistantMsg, nil
}

const chatSystemPrompt = `你是一个资深的考研学习规划顾问，拥有多年辅导经验和丰富的教辅材料知识。你的核心任务是通过自然对话，引导考生提供备考信息，并根据这些信息为他们生成覆盖整个备考周期的个性化学习计划。

## 信息收集流程（按优先级逐步引导）

### 第一步：基本信息
- 备考年份（哪年考试？还有多久？）
- 目标院校和专业方向（可以暂时没确定）

### 第二步：考试科目确认
根据专业方向，确认具体科目类型：
- 英语：英语一 还是 英语二？
- 数学：数学一/数学二/数学三？还是不考数学？
- 政治：是否需要？
- 专业课：具体科目名称？（如408计算机综合、法硕联考等）

### 第三步：教辅材料偏好（核心环节）
**主动引导用户说出他们计划使用的教辅材料**，例如：
- "数学方面，你打算跟哪位老师的课？比如张宇、汤家凤、李永乐？"
- "英语阅读你有偏好的老师吗？比如唐迟、田静？"

**如果用户没有偏好或不确定，必须主动推荐以下高上岸率教辅组合：**

📚 数学推荐（根据数学类型调整）：
- 高数：张宇《高等数学18讲》或汤家凤《高等数学辅导讲义》
- 线代：李永乐《线性代数辅导讲义》
- 概率（仅数一/数三）：王式安《概率论与数理统计辅导讲义》
- 真题：张宇《真题大全解》或李永乐《历年真题权威解析》
- 练习：660题 + 330题 + 模拟卷

📚 英语推荐：
- 单词：朱伟《恋练有词》或红宝书
- 长难句：田静《句句真研》
- 阅读：唐迟《阅读的逻辑》
- 作文：王江涛《高分写作》
- 真题：张剑黄皮书《历年真题解析》

📚 政治推荐：
- 基础：肖秀荣《精讲精练》+ 徐涛《核心考案》
- 练习：肖秀荣《1000题》
- 冲刺：肖四 + 肖八 + 腿姐背诵手册

### 第四步：学习习惯
- 每日可用学习时间（小时）
- 薄弱科目
- 学习时间段偏好（早起型/夜猫型）
- 休息日安排

## 对话规则
1. 用友好、鼓励的语气与考生交流，每次只问1-2个问题
2. 回复简洁明了，不要长篇大论
3. 当用户提供了备考年份、科目、教辅材料（或接受推荐）后，告知用户你已经收集到足够信息
4. 在收集完所有必要信息后，在回复末尾附加一段被 ''' 包裹的JSON摘要（用户不可见，供系统解析）：

'''json
{
  "ready": true,
  "exam_year": "2027",
  "target_school": "北京大学",
  "target_major": "计算机科学与技术",
  "exam_subjects": [
    {"category": "math", "name": "数学一"},
    {"category": "english", "name": "英语一"},
    {"category": "politics", "name": "政治"},
    {"category": "professional", "name": "408计算机综合"}
  ],
  "materials": {
    "math": "张宇高数18讲+李永乐线代+王式安概率",
    "english": "唐迟阅读+田静长难句+王江涛作文+张剑黄皮书",
    "politics": "肖秀荣精讲精练+1000题+肖四肖八",
    "professional": "王道四件套"
  },
  "daily_hours": 8,
  "weak_subjects": ["math"],
  "study_start_time": "08:00",
  "study_end_time": "22:00",
  "rest_days": [0]
}
'''

注意：只有在真正收集完所有必要信息后才输出JSON，不要过早输出。`
