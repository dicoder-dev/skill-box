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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------
// v2 错误(单对话文件粒度,2026-07-14 v2 引入;v1 通用 errors 仍兼容)。
// ---------------------------------------------------------------------

// ErrInvalidConvID conv_id 不合法(空 / 含 / 等不安全字符),防目录穿越(2026-07-14 增)。
var ErrInvalidConvID = errors.New("skillboxdata: invalid conv_id")

// ErrConvTooLarge 单对话文件超过 MaxConvSize,上层返 400(2026-07-14 增)。
var ErrConvTooLarge = errors.New("skillboxdata: conv file exceeds MaxConvSize")

// MaxConvSize 单个对话文件上限 2MB(2026-07-14 增,v1 是全局 5MB)。
// 单对话很少能撑到这个量,主要是防止恶意/异常超大输入。
const MaxConvSize = 2 * 1024 * 1024

// HistoryDir 单对话文件目录(2026-07-14 增 v2)。
// v2 把"一个对话 = 一个文件"放进这里;.skill-box/history.json(v1 单文件)保留作 legacy,
// 但前端已切到 v2 接口,旧文件不会被新代码读也不会被新代码生成。
const HistoryDir = "history"

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

// =====================================================================
// v2:每个对话 = 单文件(2026-07-14 增)
//
// 取代 v1 的"全部历史塞一份 .skill-box/history.json"。
// 设计动机:
//   - 一个对话 = 一份 .skill-box/history/<conv-id>.json,完整上下文自包含;
//   - 列表 API 只返 metadata 不读 messages,带宽友好;
//   - 单条 upsert / 删除粒度细,不串扰;
//
// 兼容:v1 函数(HistoryFile / ReadHistory / WriteHistory)保留未删;
// 服务层已切到 v2,旧接口走 legacy,前端不再调。
// =====================================================================

// ConvMeta 对话元数据(metadata-only,供列表展示)。
type ConvMeta struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Preview  string `json:"preview"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Ts       int64  `json:"ts"`
	Size     int64  `json:"size"`
}

// sanitizeConvID 严格白名单 [A-Za-z0-9_-],其它任何字符(包括 . / / / 控制字符)都拒,防目录穿越(2026-07-14 增)。
func sanitizeConvID(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidConvID)
	}
	// 单段最大长度,防 fat 文件名
	if len(id) > 128 {
		return "", fmt.Errorf("%w: too long", ErrInvalidConvID)
	}
	for _, r := range id {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return "", fmt.Errorf("%w: contains %q", ErrInvalidConvID, r)
		}
	}
	return id, nil
}

// ConvFile 拼出 <skillDir>/.skill-box/history/<safe-conv-id>.json 绝对路径。
// convID 不合法返 ErrInvalidConvID(不写盘外)。
func ConvFile(skillDir, convID string) (string, error) {
	safe, err := sanitizeConvID(convID)
	if err != nil {
		return "", err
	}
	return filepath.Join(Dir(skillDir), HistoryDir, safe+".json"), nil
}

// ListConvs 列出 .skill-box/history/ 下全部对话的 metadata,不读 messages(2026-07-14 增)。
//
// 行为:
//   - 目录不存在 → 返空 slice + nil error(空历史合法);
//   - 单文件损坏 / JSON 解析失败 → skip(整体不失败);
//   - 缺失 ID 字段 → 用文件名去后缀顶替;
//   - 按 ts desc 排(新的在前)。
func ListConvs(skillDir string) ([]ConvMeta, error) {
	dir := filepath.Join(Dir(skillDir), HistoryDir)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConvMeta{}, nil
		}
		return nil, fmt.Errorf("skillboxdata: readdir %s: %w", dir, err)
	}
	out := make([]ConvMeta, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var meta struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Preview  string `json:"preview"`
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Ts       int64  `json:"ts"`
		}
		if err := json.Unmarshal(b, &meta); err != nil {
			continue // 坏文件跳过,不整体失败
		}
		if meta.ID == "" {
			meta.ID = strings.TrimSuffix(e.Name(), ".json")
		}
		var size int64
		if fi, err := e.Info(); err == nil {
			size = fi.Size()
		}
		out = append(out, ConvMeta{
			ID:       meta.ID,
			Title:    meta.Title,
			Preview:  meta.Preview,
			Provider: meta.Provider,
			Model:    meta.Model,
			Ts:       meta.Ts,
			Size:     size,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ts > out[j].Ts })
	return out, nil
}

// ReadConv 读单条对话完整(2026-07-14 增)。
// 不存在返 (nil, nil),让上层判 404(source_path 校验由 service 层做)。
func ReadConv(skillDir, convID string) (*HistoryItem, error) {
	p, err := ConvFile(skillDir, convID)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("skillboxdata: read %s: %w", p, err)
	}
	var item HistoryItem
	if err := json.Unmarshal(b, &item); err != nil {
		return nil, fmt.Errorf("skillboxdata: parse %s: %w", p, err)
	}
	return &item, nil
}

// WriteConv 写单条对话,upsert;超 MaxConvSize 返 ErrConvTooLarge(2026-07-14 增)。
//
// 流程:算 preview(若 caller 没填)→ 序列化 → 检 size → atomic rename
func WriteConv(skillDir string, item HistoryItem) error {
	if _, err := sanitizeConvID(item.ID); err != nil {
		return err
	}
	if err := Ensure(skillDir); err != nil {
		return err
	}
	// v2 还需要 history/ 子目录(write path 可能在 Ensure 只建了 .skill-box/ 后立刻调,
	// 而 ListConvs 找不到目录会返空,但写这一步需要目录存在)。
	p, err := ConvFile(skillDir, item.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("skillboxdata: mkdir history dir: %w", err)
	}
	// 算 preview(若没填)
	if item.Preview == "" {
		item.Preview = previewFromMessages(item.Messages)
	}
	b, err := json.MarshalIndent(&item, "", "  ")
	if err != nil {
		return fmt.Errorf("skillboxdata: marshal: %w", err)
	}
	if len(b) > MaxConvSize {
		return fmt.Errorf("%w: %d > %d bytes", ErrConvTooLarge, len(b), MaxConvSize)
	}
	return atomicWrite(p, b)
}

// DeleteConv 删单条;不存在不报错(幂等,2026-07-14 增)。
func DeleteConv(skillDir, convID string) error {
	p, err := ConvFile(skillDir, convID)
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("skillboxdata: remove %s: %w", p, err)
	}
	return nil
}
