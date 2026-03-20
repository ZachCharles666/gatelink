package audit

import (
	"regexp"
	"strings"
)

// RiskLevel 风险等级
type RiskLevel int

const (
	RiskNone     RiskLevel = 0 // 安全
	RiskLow      RiskLevel = 1 // 低风险（需人工复查）
	RiskMedium   RiskLevel = 2 // 中风险（限制功能）
	RiskHigh     RiskLevel = 3 // 高风险（拒绝请求）
	RiskCritical RiskLevel = 4 // 极高风险（立即拒绝 + 告警）
)

// ClassifyResult 分类结果
type ClassifyResult struct {
	Level   RiskLevel
	Reason  string
	Details []string // 命中的规则列表
}

// promptInjectionPatterns Prompt 注入检测正则表达式
var promptInjectionPatterns = []*regexp.Regexp{
	// 常见注入尝试：忽略/覆盖系统提示
	regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|above|prior|your)?\s*(instructions?|prompts?|rules?|constraints?)`),
	regexp.MustCompile(`(?i)forget\s+(everything|all|what)\s+(you\s+)?(were\s+|have\s+)?(told|taught|instructed|given|said)`),
	regexp.MustCompile(`(?i)you\s+are\s+now\s+(a\s+)?new\s+(ai|model|system|assistant)`),
	// 角色扮演绕过
	regexp.MustCompile(`(?i)pretend\s+(you\s+are|to\s+be|you're)\s+(an?\s+)?(evil|unfiltered|jailbroken|dan)`),
	regexp.MustCompile(`(?i)\bDAN\b.*\bdo\s+anything\s+now\b`),
	// 系统提示泄露
	regexp.MustCompile(`(?i)(print|reveal|show|output|display)\s+(your\s+)?(system\s+prompt|instructions?|configuration)`),
	// 指令覆盖
	regexp.MustCompile(`(?i)new\s+(instructions?|rules?|directives?):\s*\n`),
	regexp.MustCompile(`(?i)###\s*(system|instruction|override)`),
	// 中文注入
	regexp.MustCompile(`忽略.*所有.*指令`),
	regexp.MustCompile(`忘记.*系统.*提示`),
	regexp.MustCompile(`现在你是.*没有限制`),
}

// Classifier Prompt 注入和风险分类器
type Classifier struct {
	filter *Filter
}

func NewClassifier(filter *Filter) *Classifier {
	return &Classifier{filter: filter}
}

// Classify 对完整对话内容进行风险分类
func (c *Classifier) Classify(messages []string) *ClassifyResult {
	result := &ClassifyResult{Level: RiskNone}
	fullText := strings.Join(messages, "\n")

	// 1. 关键词命中检测（极高风险）
	if match := c.filter.Match(fullText); match.Matched {
		return &ClassifyResult{
			Level:   RiskCritical,
			Reason:  "keyword match",
			Details: []string{"matched: " + match.Keyword},
		}
	}

	// 2. Prompt 注入检测（高风险）
	for _, pattern := range promptInjectionPatterns {
		if pattern.MatchString(fullText) {
			return &ClassifyResult{
				Level:   RiskHigh,
				Reason:  "prompt injection detected",
				Details: []string{pattern.String()},
			}
		}
	}

	// 3. 低风险规则：超长单条消息（可能的 payload 填充）
	for _, msg := range messages {
		if len([]rune(msg)) > 50000 {
			result.Level = RiskLow
			result.Reason = "oversized message"
			result.Details = append(result.Details, "message length exceeds 50000 chars")
		}
	}

	return result
}

// IsSafe 快捷方法：是否完全安全（可直接放行）
func (r *ClassifyResult) IsSafe() bool {
	return r.Level == RiskNone
}

// ShouldBlock 是否应拦截请求（高风险及以上）
func (r *ClassifyResult) ShouldBlock() bool {
	return r.Level >= RiskHigh
}
