package util

import (
	"fmt"
	"go-api-backend/internal/types"
	"strconv"
)

func BuildPrompt(query string, hits []types.Hit) string {
	promptTemplate := fmt.Sprintf("问题：%s\n\n【资料】\n", query)
	for i, hit := range hits {
		promptTemplate += fmt.Sprintf("[%s] %s\n", strconv.Itoa(i+1), hit.Text)
	}
	return promptTemplate
}
