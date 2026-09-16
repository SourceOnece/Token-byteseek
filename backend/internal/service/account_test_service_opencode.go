package service

import "github.com/gin-gonic/gin"

// OpenCode 只检测所选模型的实际协议，不把同一模型轮流发往三个互斥端点。
func (s *AccountTestService) testOpenCodeAccountConnection(c *gin.Context, account *Account, model, prompt string) error {
	return s.testCNProviderAccountConnection(c, account, model, prompt)
}
