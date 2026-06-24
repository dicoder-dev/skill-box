// Package skillstore 实现 canonical skill 的物理存储。
//
// 目录布局(对应 StoreRoot,默认 ~/.skill-box/skills,贴合 Claude Code 风格):
//
//	<StoreRoot>/<name>/SKILL.md
//	<StoreRoot>/<name>/...
//
// 设计要点:
//   - 一个 skill 一个目录,无 version 层(版本写在 SKILL.md frontmatter)
//   - 元数据唯一来源是 SKILL.md 的 YAML frontmatter,不再额外落 skill.yaml
//   - 写入走 per-skill 文件锁(flock),保证多进程并发安全
//   - 跨工具兼容:任何按 "<name>/SKILL.md" 布局的外部工具(Claude Code / Codex / ...)
//     都可以直接读本目录;我们要写回时也只动 SKILL.md,不会引入额外元数据文件
//
// 设计上下文见 docs/project/需求规划.md 第 5.1 + 8.2 节。
package skillstore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"ginp-api/configs"
	"ginp-api/internal/skilladapter"
	sharefunc "ginp-api/share/func"
)

// ErrNotFound skill 不存在。
var ErrNotFound = errors.New("skillstore: not found")

// Store canonical skill 物理存储。
type Store struct {
	root string
}

// New 根据配置构造 Store;StoreRoot 为空时使用 ~/.skill-box/skills 兜底。
func New() (*Store, error) {
	if root := strings.TrimSpace(configs.Skillbox.StoreRoot); root != "" {
		return NewAt(root)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("skillstore: cannot resolve home dir: %w", err)
	}
	return NewAt(filepath.Join(home, ".skill-box", "skills"))
}

// NewAt 显式指定 root,主要用于测试。
func NewAt(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("skillstore: mkdir root %s: %w", root, err)
	}
	return &Store{root: root}, nil
}

// Root 返回当前 store 根目录。
func (s *Store) Root() string { return s.root }

// DataDir 返回应用主数据目录(~/.<AppName>/,默认 ~/.skill-box),便于 caller
// 把日志、数据库等其它数据放在同一棵树下。
func (s *Store) DataDir() string {
	if s == nil {
		return sharefunc.DataDir()
	}
	// 从 root 向上回溯两级:skills → .skill-box
	parent := filepath.Dir(s.root)
	if filepath.Base(parent) != "skills" {
		return sharefunc.DataDir()
	}
	return filepath.Dir(parent)
}

// HashFile 计算单文件 SHA-256 摘要(hex)。
func HashFile(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// Save 写入 canonical skill(覆盖式)。
// 写入流程:加文件锁 → 写临时目录 → 原子 rename → 释放锁。
//
// 无 version 目录:直接把整个 Canonical.Files 写进 root/<name>/。
// SKILL.md 是必填(由 WriteSkillDir 强校验),其它附属文件照原样铺平。
func (s *Store) Save(c skilladapter.Canonical) error {
	if strings.TrimSpace(c.Manifest.Name) == "" {
		return fmt.Errorf("skillstore: name is empty")
	}
	dir := s.skillDir(c.Manifest.Name)

	unlock, err := s.lockScope(dir)
	if err != nil {
		return err
	}
	defer unlock()

	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".skill-tmp-*")
	if err != nil {
		return fmt.Errorf("skillstore: mkdir temp: %w", err)
	}
	defer os.RemoveAll(tmp)

	// 写文件 — SKILL.md 必须包含 frontmatter,所以无论 caller 是否已带
	// SKILL.md 字段,这里都用 RenderSkillMD 重新渲染一份,保证 frontmatter
	// 一定存在且与 Manifest 一致。
	if err := writeFileAtomic(filepath.Join(tmp, "SKILL.md"), skilladapter.RenderSkillMD(c), 0o644); err != nil {
		return err
	}
	for _, f := range c.Files {
		if f.Path == "" || f.Path == "SKILL.md" {
			continue
		}
		rel, err := safeRelPath(f.Path)
		if err != nil {
			return fmt.Errorf("skillstore: invalid file path %q: %w", f.Path, err)
		}
		dst := filepath.Join(tmp, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("skillstore: mkdir %s: %w", filepath.Dir(dst), err)
		}
		if err := writeFileAtomic(dst, f.Content, 0o644); err != nil {
			return err
		}
	}

	// 原子替换:先把目标目录(如果存在)删了,再 rename temp -> target
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("skillstore: remove old dir: %w", err)
	}
	if err := os.Rename(tmp, dir); err != nil {
		return fmt.Errorf("skillstore: rename temp: %w", err)
	}
	return nil
}

// Load 读取 canonical skill;不存在返回 (nil, ErrNotFound)。
// 单一来源是 SKILL.md 的 frontmatter + 同目录附属文件。
func (s *Store) Load(name string) (*skilladapter.Canonical, error) {
	dir := s.skillDir(name)
	skillMD := filepath.Join(dir, "SKILL.md")
	content, err := os.ReadFile(skillMD)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("skillstore: read SKILL.md: %w", err)
	}
	c, err := skilladapter.ParseSkillMD(string(content))
	if err != nil {
		return nil, fmt.Errorf("skillstore: parse SKILL.md: %w", err)
	}
	// 强制把 name 锚定到目录(避免外部 SKILL.md 改 name 漂移)
	c.Manifest.Name = name
	// 把同名文件塞回去(已含 SKILL.md);其它附属文件一并加载
	c.Files, err = walkFiles(dir)
	if err != nil {
		return nil, err
	}
	// 兜底:解析失败时 frontmatter 给的 files 列表可能没有 SKILL.md
	hasMain := false
	for _, f := range c.Files {
		if f.Path == "SKILL.md" {
			hasMain = true
			break
		}
	}
	if !hasMain {
		c.Files = append([]skilladapter.File{{Path: "SKILL.md", Content: string(content)}}, c.Files...)
	}
	return c, nil
}

// Delete 删除 skill(整个目录)。缺失时返回 nil(幂等)。
func (s *Store) Delete(name string) error {
	dir := s.skillDir(name)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(dir)
	_ = removeIfEmpty(parent)
	return nil
}

// Exists 判断指定 skill 是否存在(有 SKILL.md 就算存在)。
func (s *Store) Exists(name string) bool {
	dir := s.skillDir(name)
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

// List 列出全部 skill 的 Canonical(目录扫描 + frontmatter 解析)。
// 损坏的 skill 跳过,不阻塞整体;keyword 非空时做 name 子串匹配(不区分大小写)。
func (s *Store) List(keyword string) ([]skilladapter.Canonical, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []skilladapter.Canonical
	kw := strings.ToLower(strings.TrimSpace(keyword))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			// 隐藏目录视为非 skill(避免 .system / .curated 这类系统子目录混入)
			continue
		}
		if _, err := os.Stat(filepath.Join(s.root, name, "SKILL.md")); err != nil {
			continue
		}
		c, err := s.Load(name)
		if err != nil {
			// 损坏的 skill 跳过,不让一个坏文件搞挂全表
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(c.Manifest.Name), kw) {
			continue
		}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.Name < out[j].Manifest.Name })
	return out, nil
}

// --- internals ---

// skillDir 计算某个 skill 的实际目录。无 version 层。
func (s *Store) skillDir(name string) string {
	return filepath.Join(s.root, name)
}

// writeFileAtomic 先写临时文件再 rename,避免半截文件。
func writeFileAtomic(path, content string, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".skill-f-*")
	if err != nil {
		return fmt.Errorf("skillstore: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // 如果 rename 成功这里会失败,无害

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("skillstore: write temp file: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("skillstore: rename temp file: %w", err)
	}
	return nil
}

// walkFiles 递归扫目录里所有文件(用于 Load 时取附属文件)。
func walkFiles(root string) ([]skilladapter.File, error) {
	var files []skilladapter.File
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, skilladapter.File{
			Path:    filepath.ToSlash(rel),
			Content: string(content),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("skillstore: walk %s: %w", root, err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// safeRelPath 拒绝 ..、绝对路径、含 \0 等可疑 path。
func safeRelPath(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("absolute path not allowed")
	}
	if strings.Contains(p, "\x00") {
		return "", fmt.Errorf("path contains NUL")
	}
	cleaned := filepath.Clean(p)
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal not allowed")
	}
	return cleaned, nil
}

func removeIfEmpty(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}

// fileLock 简单的 per-scope 文件锁。同一进程内用 mutex 兜底,
// 跨进程靠 flock(系统调用)。
type fileLock struct {
	path string
	f    *os.File
	mu   *sync.Mutex
}

var inprocLocks sync.Map // path -> *sync.Mutex

func (s *Store) lockScope(dir string) (func(), error) {
	lockPath := dir + ".lock"
	v, _ := inprocLocks.LoadOrStore(lockPath, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()

	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		mu.Unlock()
		return nil, fmt.Errorf("skillstore: open lock: %w", err)
	}
	if err := flock(f); err != nil {
		f.Close()
		mu.Unlock()
		return nil, fmt.Errorf("skillstore: flock: %w", err)
	}

	unlocked := false
	unlock := func() {
		if unlocked {
			return
		}
		unlocked = true
		_ = funlock(f)
		f.Close()
		mu.Unlock()
		// 清理 .lock 临时文件(留 root 看着像脏)
		_ = os.Remove(lockPath)
	}
	return unlock, nil
}
