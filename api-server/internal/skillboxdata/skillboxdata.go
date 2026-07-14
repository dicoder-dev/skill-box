// Package skillboxdata 管理每个 skill 目录下 ".skill-box/" 运行时数据目录。
//
// 用途:存放 skillbox 应用产生的、与 skill 自身内容无关的私有数据。
// 当前主要是 chat history,后续可扩展 caches / notes 等。
//
// 写盘:前端 AI 面板双写到本地 .skill-box/history.json;
// 读盘:前端 AI 面板打开"历史对话"Modal 时从后端拉远端列表。
//
// 不会出现在 c.Files:本目录由独立 service 直管,
// skillstore.walkFiles / skilladapter.readDirFiles / BaseAdapter.Apply
// 都已经把 ".skill-box/" / 任何 ". 开头的段"过滤掉(2026-07-14 起)。
//
// 不会喂给 AI:见 (skilltester).ai_walker.go#buildSkillMDForPrompt 与
// (caiprovider).chat_stream.a.go#guardSkillBoxSystem。
package skillboxdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 目录与文件名。
const (
	// DirName 隐藏目录名(与 skillstore 既有 walkFiles 过滤规则对齐)。
	DirName = ".skill-box"

	// ReadmeFile 固定说明文件,Ensure() 一次性写入(已存在则不动)。
	ReadmeFile = "readme.md"

	// HistoryFile 历史对话 JSON。
	HistoryFile = "history.json"

	// MaxHistorySize 单个 .skill-box/history.json 字节上限,
	// 超过按 ts 升序 FIFO 截断。
	MaxHistorySize = 5 * 1024 * 1024

	// HistoryVersion schema 版本。前端读取后会按 version 决定解析方式。
	HistoryVersion = 1
)

// readmeContent 写到 .skill-box/readme.md 的固定内容。
//
// 这段文字面向开发者(人),不是给 AI 读的(AI 路径已经在 ai_walker + chat_stream
// 双重过滤掉 .skill-box/ 了)。它存在意义:
//   - 解释这个目录用途,避免有人手动删;
//   - 给将来若我们放开 AI 访问时提供一段"安全上下文"。
var readmeContent = []byte(`# .skill-box/

This directory is owned by **Skill Box** and stores runtime data
that the application generates for this skill (chat history,
caches, etc.).

**Do not read this directory**:

- AI assistants must ignore any file under '.skill-box/'.
- Tools must not enumerate, parse, or surface it.
- Editing files here manually is unsupported — your changes may be overwritten.

If you are the skill author and want to ship documentation or
runtime assets with your skill, place them alongside 'SKILL.md'
instead of inside '.skill-box/'.
`)

// Dir 返回 skillDir 下 .skill-box/ 的绝对路径(不保证存在)。
func Dir(skillDir string) string {
	return filepath.Join(skillDir, DirName)
}

// Ensure 创建 .skill-box/ 并写入 readme.md(readme 已存在则不动)。
// 幂等:对已存在的目录也安全。
func Ensure(skillDir string) error {
	dir := Dir(skillDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("skillboxdata: mkdir %s: %w", dir, err)
	}
	rm := filepath.Join(dir, ReadmeFile)
	if _, err := os.Stat(rm); os.IsNotExist(err) {
		if err := os.WriteFile(rm, readmeContent, 0o644); err != nil {
			return fmt.Errorf("skillboxdata: write %s: %w", rm, err)
		}
	} else if err != nil {
		return fmt.Errorf("skillboxdata: stat %s: %w", rm, err)
	}
	return nil
}

// HistoryItem 单条对话记录。
// Messages 用 json.RawMessage 接,避免让 history 包反向依赖 blades 的 message 类型。
type HistoryItem struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Preview  string          `json:"preview"`
	Ts       int64           `json:"ts"`
	Provider string          `json:"provider,omitempty"`
	Model    string          `json:"model,omitempty"`
	Messages json.RawMessage `json:"messages"`
}

// History history.json 顶层结构。
type History struct {
	Version int          `json:"version"`
	Items   []HistoryItem `json:"items"`
}

// ReadHistory 读取 .skill-box/history.json;
// 不存在返 &History{Version: HistoryVersion},不算 error(空历史合法)。
func ReadHistory(skillDir string) (*History, error) {
	p := filepath.Join(Dir(skillDir), HistoryFile)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &History{Version: HistoryVersion, Items: []HistoryItem{}}, nil
		}
		return nil, fmt.Errorf("skillboxdata: read %s: %w", p, err)
	}
	var h History
	if err := json.Unmarshal(b, &h); err != nil {
		return nil, fmt.Errorf("skillboxdata: parse %s: %w", p, err)
	}
	if h.Version == 0 {
		h.Version = HistoryVersion
	}
	if h.Items == nil {
		h.Items = []HistoryItem{}
	}
	return &h, nil
}

// WriteHistory 写入 .skill-box/history.json,带容量截断与 atomic rename。
//
// 容量截断规则:序列化后的字节数若超过 MaxHistorySize,按 ts 升序删到 ≤ 上限。
// 截断发生在序列化前(更准确 — 用 Marshal 后 byte slice len 判断)。
func WriteHistory(skillDir string, items []HistoryItem) error {
	if err := Ensure(skillDir); err != nil {
		return err
	}
	h := &History{Version: HistoryVersion, Items: items}

	// 算 preview:从 messages 抽出首条 assistant content 的前 120 字。
	for i := range h.Items {
		if h.Items[i].Preview == "" {
			h.Items[i].Preview = previewFromMessages(h.Items[i].Messages)
		}
	}

	// 序列化 + 容量截断(FIFO,按 ts 升序删旧)。
	const maxRetry = 16
	for attempt := 0; attempt < maxRetry; attempt++ {
		b, err := json.MarshalIndent(h, "", "  ")
		if err != nil {
			return fmt.Errorf("skillboxdata: marshal: %w", err)
		}
		if len(b) <= MaxHistorySize || len(h.Items) <= 1 {
			return atomicWrite(filepath.Join(Dir(skillDir), HistoryFile), b)
		}
		// 按 ts 升序,删最旧的
		sort.SliceStable(h.Items, func(i, j int) bool { return h.Items[i].Ts < h.Items[j].Ts })
		// 至少保留最新一条(避免空)
		h.Items = h.Items[1:]
	}
	return fmt.Errorf("skillboxdata: history still > %d bytes after %d retries", MaxHistorySize, maxRetry)
}

// previewFromMessages 简单预览算法:从 messages 数组里找首条 assistant,
// 取前 120 字作为预览。失败返空串。
func previewFromMessages(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var arr []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return ""
	}
	for _, m := range arr {
		if m.Role == "assistant" {
			s := strings.TrimSpace(m.Content)
			runes := []rune(s)
			if len(runes) > 120 {
				return string(runes[:120]) + "…"
			}
			return s
		}
	}
	return ""
}

// atomicWrite 写到临时文件 + rename,避免半截写入损坏 history.json。
func atomicWrite(p string, b []byte) error {
	dir := filepath.Dir(p)
	tmp, err := os.CreateTemp(dir, ".skill-box-history-*.tmp")
	if err != nil {
		return fmt.Errorf("skillboxdata: create temp: %w", err)
	}
	tmpName := tmp.Name()
	// 清理:出错时删 tmp;成功时由 rename 接管。
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("skillboxdata: write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("skillboxdata: sync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("skillboxdata: close tmp: %w", err)
	}
	if err := os.Rename(tmpName, p); err != nil {
		cleanup()
		return fmt.Errorf("skillboxdata: rename: %w", err)
	}
	return nil
}
