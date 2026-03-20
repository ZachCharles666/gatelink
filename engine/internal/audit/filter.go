// Package audit 提供内容审核功能。
// 设计原则：快路径（关键词命中）在本地完成，无需外部 API 调用。
package audit

import (
	"strings"
	"sync"
	"unicode"
)

// defaultKeywords 内置违禁关键词（中英文）
// 生产环境应从数据库或配置文件热加载，此为 MVP 内置词库
var defaultKeywords = []string{
	// 暴力
	"炸弹制作", "爆炸物", "如何制造武器", "homemade bomb", "make explosive",
	// 钓鱼/欺诈
	"钓鱼链接", "phishing", "social engineering", "credential harvesting",
	// 恶意代码
	"恶意软件", "勒索病毒", "ransomware", "malware payload", "keylogger",
	// 隐私侵犯
	"人肉搜索", "doxxing",
	// 其他高风险
	"CSAM", "child exploitation",
}

// Filter 关键词过滤器，支持热更新
type Filter struct {
	mu       sync.RWMutex
	keywords []string
}

// NewFilter 创建过滤器（使用内置词库）
func NewFilter() *Filter {
	f := &Filter{}
	f.keywords = make([]string, len(defaultKeywords))
	copy(f.keywords, defaultKeywords)
	return f
}

// Reload 热更新关键词列表（线程安全）
func (f *Filter) Reload(keywords []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keywords = make([]string, len(keywords))
	copy(f.keywords, keywords)
}

// AddKeywords 追加关键词（不替换现有词库）
func (f *Filter) AddKeywords(words []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keywords = append(f.keywords, words...)
}

// MatchResult 关键词命中结果
type MatchResult struct {
	Matched bool
	Keyword string // 命中的关键词（脱敏显示，只取前 3 字/字符）
}

// Match 检测文本是否命中违禁关键词
// 大小写不敏感，忽略多余空白
func (f *Filter) Match(text string) MatchResult {
	f.mu.RLock()
	defer f.mu.RUnlock()

	normalized := normalizeText(text)
	for _, kw := range f.keywords {
		if strings.Contains(normalized, normalizeText(kw)) {
			return MatchResult{
				Matched: true,
				Keyword: maskKeyword(kw),
			}
		}
	}
	return MatchResult{Matched: false}
}

// normalizeText 统一化文本：转小写、去除多余空白
func normalizeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// maskKeyword 对命中的关键词进行脱敏，只返回前 3 个字符/汉字
func maskKeyword(kw string) string {
	runes := []rune(kw)
	if len(runes) <= 3 {
		return "***"
	}
	return string(runes[:3]) + "***"
}
