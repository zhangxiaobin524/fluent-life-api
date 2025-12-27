package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAIService(db *gorm.DB, cfg *config.Config) *AIService {
	return &AIService{db: db, cfg: cfg}
}

type GeminiMessage struct {
	Role string `json:"role"`
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}

type GeminiRequest struct {
	Contents []GeminiMessage `json:"contents"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (s *AIService) Chat(userID uuid.UUID, userMessage string) (string, error) {
	// 获取或创建对话记录
	var conversation models.AIConversation
	err := s.db.Where("user_id = ?", userID).First(&conversation).Error
	if err == gorm.ErrRecordNotFound {
		conversation = models.AIConversation{
			UserID:   userID,
			Messages: models.Messages{},
		}
		s.db.Create(&conversation)
	}

	// 添加用户消息
	conversation.Messages = append(conversation.Messages, models.Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      "user",
		Text:      userMessage,
		Timestamp: time.Now(),
	})

	// 调用 Gemini API
	responseText, err := s.callGeminiAPI(conversation.Messages)
	if err != nil {
		return "", err
	}

	// 添加AI回复
	conversation.Messages = append(conversation.Messages, models.Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      "bot",
		Text:      responseText,
		Timestamp: time.Now(),
	})

	// 更新对话记录
	conversation.UpdatedAt = time.Now()
	s.db.Save(&conversation)

	return responseText, nil
}

func (s *AIService) callGeminiAPI(messages models.Messages) (string, error) {
	if s.cfg.GeminiAPIKey == "" {
		// 开发环境返回模拟回复
		mocks := []string{
			"这是一个很好的观察。记住，每一次卡顿都是身体在提醒你慢下来，尝试用'软起音'开始下一句。",
			"我听到了你的进步！在刚才的话语中，你成功地保持了气流的连贯。",
			"不要急于摆脱口吃，试着去观察它发生时肌肉的张力。我们一起练习深呼吸。",
			"心态的转变比技巧更重要。允许自己卡顿，这份从容会带给你真正的流畅。",
		}
		return mocks[time.Now().Unix()%int64(len(mocks))], nil
	}

	// 构建请求
	var geminiMessages []GeminiMessage
	for _, msg := range messages {
		geminiMessages = append(geminiMessages, GeminiMessage{
			Role: msg.Role,
			Parts: []struct {
				Text string `json:"text"`
			}{{Text: msg.Text}},
		})
	}

	reqBody := GeminiRequest{Contents: geminiMessages}
	jsonData, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=%s", s.cfg.GeminiAPIKey)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("API返回格式错误")
}

func (s *AIService) GetConversation(userID uuid.UUID) (*models.AIConversation, error) {
	var conversation models.AIConversation
	err := s.db.Where("user_id = ?", userID).First(&conversation).Error
	if err == gorm.ErrRecordNotFound {
		return &models.AIConversation{
			UserID:   userID,
			Messages: models.Messages{},
		}, nil
	}
	return &conversation, err
}

func (s *AIService) AnalyzeSpeech(transcription string) (string, error) {
	// 模拟分析
	analysis := fmt.Sprintf(`### 模拟分析报告
  
- **发音张力**：中等。在起首音处有明显的喉部肌肉紧缩。
- **停顿节奏**：良好。你已经开始有意识地在句中留白。
- **改进建议**：继续加强"气流调节"模块的练习，重点关注呼气与发音的同步性。`)
	return analysis, nil
}







