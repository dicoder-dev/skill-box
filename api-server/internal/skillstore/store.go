// Package skillstore 实现 canonical skill 的物理存储。
//
// 目录布局(对应 StoreRoot,默认 ~/.skillbox/store):
//
//	<StoreRoot>/global/<name>/<version>/skill.yaml
//	<StoreRoot>/global/<name>/<version>/SKILL.md
//	<StoreRoot>/global/<name>/<version>/...
//	<StoreRoot>/project/<projectID>/<name>/<version>/...
//
// 写入走 per-skill 文件锁(flock),保证多进程并发安全。
// 设计见 docs/project/需求规划.md 第 5.1 + 8.2 节。
package skillstore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"ginp-api/configs"
	"ginp-api/internal/skilladapter"

	"gopkg.in/yaml.v3"
)

const manifestFileName = "skill.yaml"

// ErrNotFound skill 不存在。
var ErrNotFound = errors.New("skillstore: not found")

// ErrAlreadyExists skill 已存在(Save 在覆盖语义下不会返回,Update 才会)。
var ErrAlreadyExists = errors.New("skillstore: already exists")

// Store canonical skill 物理存储。
type Store struct {
	root string
}

// New 根据配置构造 Store;StoreRoot 为空时使用 OS 用户目录兜底。
func New() (*Store, error) {
	root := strings.TrimSpace(configs.Skillbox.StoreRoot)
	if root == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("skillstore: cannot resolve home dir: %w", err)
		}
		root = filepath.Join(h, ".skillbox", "store")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("skillstore: mkdir root %s: %w", root, err)
	}
	return &Store{root: root}, nil
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

// skillDir 计算某个 skill 的实际目录。
// scope=project 时 projectID 必填。
func (s *Store) skillDir(scope, name, version string, projectID uint) string {
	if scope == skilladapter.ScopeProject {
		return filepath.Join(s.root, "project", fmt.Sprintf("%d", projectID), name, version)
	}
	return filepath.Join(s.root, "global", name, version)
}

// lockPath per-skill 锁文件路径。
func (s *Store) lockPath(scope, name, version string, projectID uint) string {
	return s.skillDir(scope, name, version, projectID) + ".lock"
}

// HashFile 计算单文件 SHA-256 摘要。
func HashFile(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// Save 写入 canonical skill(覆盖式)。
// 写入流程:加文件锁 → 写临时目录 → 原子 rename → 释放锁。
func (s *Store) Save(c skilladapter.Canonical, scope string, projectID uint) error {
	if err := validateManifest(c.Manifest); err != nil {
		return err
	}
	dir := s.skillDir(scope, c.Manifest.Name, c.Manifest.Version, projectID)
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return fmt.Errorf("skillstore: mkdir %s: %w", filepath.Dir(dir), err)
	}

	unlock, err := s.lockScope(dir)
	if err != nil {
		return err
	}
	defer unlock()

	// 准备临时目录
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".skill-tmp-*")
	if err != nil {
		return fmt.Errorf("skillstore: mkdir temp: %w", err)
	}
	defer os.RemoveAll(tmp)

	// 写 manifest
	manifestBytes, err := yaml.Marshal(c.Manifest)
	if err != nil {
		return fmt.Errorf("skillstore: marshal manifest: %w", err)
	}
	if err := writeFileAtomic(filepath.Join(tmp, manifestFileName), string(manifestBytes), 0o644); err != nil {
		return err
	}

	// 写其它文件
	for _, f := range c.Files {
		if f.Path == "" {
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
	// 注意:rename 在 Linux 上若目标存在会失败;macOS 同理
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("skillstore: remove old dir: %w", err)
	}
	if err := os.Rename(tmp, dir); err != nil {
		return fmt.Errorf("skillstore: rename temp: %w", err)
	}
	return nil
}

// Load 读取 canonical skill;不存在返回 (nil, ErrNotFound)。
func (s *Store) Load(scope, name, version string, projectID uint) (*skilladapter.Canonical, error) {
	dir := s.skillDir(scope, name, version, projectID)
	manifest, err := os.ReadFile(filepath.Join(dir, manifestFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("skillstore: read manifest: %w", err)
	}
	var m skilladapter.Manifest
	if err := yaml.Unmarshal(manifest, &m); err != nil {
		return nil, fmt.Errorf("skillstore: parse manifest: %w", err)
	}
	files, err := walkFiles(dir)
	if err != nil {
		return nil, err
	}
	return &skilladapter.Canonical{Manifest: m, Files: files}, nil
}

// Delete 删除 skill(整个目录)。
// 缺失时返回 nil(幂等)。
func (s *Store) Delete(scope, name, version string, projectID uint) error {
	dir := s.skillDir(scope, name, version, projectID)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	// 清理空的上级
	parent := filepath.Dir(dir)
	_ = removeIfEmpty(parent)
	if scope == skilladapter.ScopeProject {
		_ = removeIfEmpty(filepath.Dir(parent))
	}
	return nil
}

// ListVersions 列出某 skill 的全部版本(按 semver 字符串排序,非语义排序)。
func (s *Store) ListVersions(scope, name string, projectID uint) ([]string, error) {
	parent := filepath.Join(s.root, scopeToSeg(scope), nameSeg(scope, projectID), name)
	entries, err := os.ReadDir(parent)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(parent, e.Name(), manifestFileName)); err == nil {
			versions = append(versions, e.Name())
		}
	}
	sort.Strings(versions)
	return versions, nil
}

// ListNames 列出某 scope 下的全部 skill 名(去重,跨版本合并)。
func (s *Store) ListNames(scope string, projectID uint) ([]string, error) {
	base := filepath.Join(s.root, scopeToSeg(scope), nameSeg(scope, projectID))
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// --- internals ---

func scopeToSeg(scope string) string {
	if scope == skilladapter.ScopeProject {
		return "project"
	}
	return "global"
}

func nameSeg(scope string, projectID uint) string {
	if scope == skilladapter.ScopeProject {
		return fmt.Sprintf("%d", projectID)
	}
	return ""
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
		if rel == manifestFileName {
			// 不重复暴露 skill.yaml 为 File;Caller 可从 Manifest 字段读
			return nil
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
	}
	return unlock, nil
}

// --- validation ---

var (
	nameRE    = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
	versionRE = regexp.MustCompile(`^v?\d+\.\d+\.\d+([-+].+)?$`)
)

func validateManifest(m skilladapter.Manifest) error {
	if !nameRE.MatchString(m.Name) {
		return fmt.Errorf("skillstore: invalid name %q (want %s)", m.Name, nameRE.String())
	}
	if !versionRE.MatchString(m.Version) {
		return fmt.Errorf("skillstore: invalid version %q", m.Version)
	}
	if l := len(m.Description); l < 10 || l > 500 {
		return fmt.Errorf("skillstore: description length %d out of [10,500]", l)
	}
	if len(m.Triggers) < 1 || len(m.Triggers) > 10 {
		return fmt.Errorf("skillstore: triggers count %d out of [1,10]", len(m.Triggers))
	}
	return nil
}
