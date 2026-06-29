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

// maxScanDepth 递归扫描的最大深度(2026-06-29 增,与 skilladapter.BaseAdapter
// 的同名常量保持一致,防止分组嵌套过深导致扫描死循环)。
const maxScanDepth = 8

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
//
// 2026-06-29 改:支持 groupPath;当 c.Manifest.GroupPath 非空时,skill 写到
// root/<groupPath>/<name>/。name 走 NormalizeName 规约(不含 '/'),
// groupPath 由 caller 走 NormalizeGroupName 规约(允许 '/')。
func (s *Store) Save(c skilladapter.Canonical) error {
	if strings.TrimSpace(c.Manifest.Name) == "" {
		return fmt.Errorf("skillstore: name is empty")
	}
	dir, err := s.resolveSkillDir(c.Manifest.GroupPath, c.Manifest.Name)
	if err != nil {
		return err
	}

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
//
// 2026-06-29 改:仍按 name 查"根下直接子目录";多级分组请用 LoadByPath。
func (s *Store) Load(name string) (*skilladapter.Canonical, error) {
	dir, err := s.skillDir(name)
	if err != nil {
		return nil, err
	}
	return s.loadFromDir(dir)
}

// LoadByPath 读取指定分组路径下的 canonical skill;不存在返回 (nil, ErrNotFound)。
//
// 2026-06-29 增:支持多级分组,groupPath 为空时等价于 Load(name)。
func (s *Store) LoadByPath(groupPath string, name string) (*skilladapter.Canonical, error) {
	dir, err := s.resolveSkillDir(groupPath, name)
	if err != nil {
		return nil, err
	}
	return s.loadFromDir(dir)
}

// loadFromDir 是 Load / LoadByPath 共用的"读目录"实现。
func (s *Store) loadFromDir(dir string) (*skilladapter.Canonical, error) {
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
	// 用目录最后一层名作为 name(避免外部 SKILL.md 改 name 漂移);
	// 同时把 GroupPath 也回填(由目录相对 root 的路径反推)。
	rel, relErr := filepath.Rel(s.root, dir)
	if relErr != nil {
		rel = ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		c.Manifest.GroupPath = ""
		c.Manifest.Name = filepath.Base(dir)
	} else {
		// 多级:GroupPath = rel 的父路径;Name = rel 的最后一层
		c.Manifest.GroupPath = filepath.Dir(rel)
		if c.Manifest.GroupPath == "." {
			c.Manifest.GroupPath = ""
		}
		c.Manifest.Name = filepath.Base(rel)
	}
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
//
// 2026-06-29 改:旧 API 仍按 name 删"根下直接子目录";多级分组请用 DeleteByPath。
func (s *Store) Delete(name string) error {
	dir, err := s.skillDir(name)
	if err != nil {
		return err
	}
	return s.deleteDir(dir)
}

// DeleteByPath 删除指定分组路径下的 skill 目录。缺失时返回 nil(幂等)。
//
// 2026-06-29 增:支持多级分组。
func (s *Store) DeleteByPath(groupPath string, name string) error {
	dir, err := s.resolveSkillDir(groupPath, name)
	if err != nil {
		return err
	}
	return s.deleteDir(dir)
}

// deleteDir 是 Delete / DeleteByPath 共用的"删目录"实现。
func (s *Store) deleteDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(dir)
	_ = removeIfEmpty(parent)
	return nil
}

// Exists 判断指定 skill 是否存在(有 SKILL.md 就算存在)。
//
// 2026-06-29 改:旧 API 仍按 name 查"根下直接子目录";多级分组请用 ExistsByPath。
func (s *Store) Exists(name string) bool {
	dir, err := s.skillDir(name)
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

// ExistsByPath 判断指定分组路径下的 skill 是否存在。
//
// 2026-06-29 增:支持多级分组。
func (s *Store) ExistsByPath(groupPath string, name string) bool {
	dir, err := s.resolveSkillDir(groupPath, name)
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

// MoveGroupPath 把 skill 从 srcGroupPath 移动到 dstGroupPath 下(叶子 name 不变)。
//
// 2026-06-29 增:支持多级分组,实现策略 —
//   - 若 source 不存在,返回 ErrNotFound
//   - 若 dstGroupPath 已存在同名 skill,返回 error(避免覆盖)
//   - 内部走 os.Rename(同设备下原子),跨设备降级为 copy+delete
// 2026-06-29 改:加 ancestor check — 若 dstDir 在 srcDir 内部,直接 400 拒掉
// (防死循环,见 MoveGroupDir 同名注释)。
//
// 注意:本函数只移动单个 skill 叶子目录;移动整个分组请用 MoveGroupDir。
func (s *Store) MoveGroupPath(srcGroupPath string, name string, dstGroupPath string) error {
	srcDir, err := s.resolveSkillDir(srcGroupPath, name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(srcDir, "SKILL.md")); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	dstDir, err := s.resolveSkillDir(dstGroupPath, name)
	if err != nil {
		return err
	}
	// 2026-06-29 增:防御性 ancestor check。dstDir 在 srcDir 内部 = 把 skill 挪到
	// 自己的子目录下,os.Rename 必失败,降级 copyDirRecursive 必死循环。
	if isDescendantOrSame(dstDir, srcDir) {
		return fmt.Errorf("skillstore: cannot move skill %q into its own descendant %q", name, dstGroupPath)
	}
	// 目标已存在 → 拒覆盖(让 caller 决定是否先删)
	if _, err := os.Stat(filepath.Join(dstDir, "SKILL.md")); err == nil {
		return fmt.Errorf("skillstore: target %q already exists", dstDir)
	}
	// 确保目标父目录存在
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("skillstore: mkdir dst %s: %w", dstDir, err)
	}
	// 跨目录 rename(同一文件系统下是原子的;跨设备会退化为 copy+delete)
	if err := os.Rename(srcDir, dstDir); err != nil {
		if cerr := copyDirRecursive(srcDir, dstDir); cerr != nil {
			return fmt.Errorf("skillstore: move failed (rename=%v, copy=%v)", err, cerr)
		}
		if rerr := os.RemoveAll(srcDir); rerr != nil {
			return fmt.Errorf("skillstore: move source cleanup failed: %w", rerr)
		}
	}
	// 清理 source 空父目录链
	srcParent := filepath.Dir(srcDir)
	_ = removeIfEmpty(srcParent)
	return nil
}

// MoveGroupDir 把整个分组目录从 srcGroupPath 移动到 dstGroupPath 下。
// dstGroupPath 可以为空(=把分组挪到根下);name 不变(取 src 的最后一段)。
//
// 2026-06-29 增:复用 MoveGroupPath 思路,作用对象是整个分组目录子树。
// 2026-06-29 改:加 ancestor check — 若 dstGroupPath 是 srcGroupPath 的祖先/自身
// (或反过来,src 在 dst 内部),直接 400 拒掉,防死循环(见 copyDirRecursive 注释)。
// 2026-06-29 再改:加 no-op 幂等处理 — 当 dstAbs == srcAbs 时(典型 case: 根下
// 分组"挪到根" src=aa,dst="" → dstAbs=root/aa=srcAbs;或 aa/bb "挪到 aa 下"
// src=aa/bb,dst=aa → dstAbs=root/aa/bb=srcAbs),目标就是当前位置,直接返 OK。
// 注释里早就说了 "src=aa,dst="" → 合法",但实现没短路,导致走到"目标已存在"判
// 断时被误拒(2026-06-29 用户报告的 "target group .../aa already exists" 就是
// 这个 case)。同位置 rename 在 os.Rename 层面是 noop,但前端会更早一步撞到
// 我们的存在性 check,所以必须在 store 层先拦。
//
// 用 group path 判 ancestor,不用 abs path,这样:
//   - src=aa,dst=""     → 合法(挪到根,目标 = root/aa,no-op 短路返 OK)
//   - src=aa,dst=aa/yy  → 非法(目标 = root/aa/yy/aa,在 src 内部,会死循环)
//   - src=aa,dst=aa     → noop 幂等返 OK(不算非法)
//   - src=aa/bb,dst=aa  → 非法(把 bb 挪到 aa 下,目标 = root/aa/bb,等于 src,no-op
//     但会引发 copyDirRecursive 自己 copy 自己)
func (s *Store) MoveGroupDir(srcGroupPath string, dstGroupPath string) error {
	if srcGroupPath == "" {
		return fmt.Errorf("skillstore: empty src group path")
	}
	srcRel, err := safeRelPath(srcGroupPath)
	if err != nil {
		return err
	}
	srcAbs := filepath.Join(s.root, filepath.FromSlash(srcRel))
	if _, err := os.Stat(srcAbs); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	srcBase := filepath.Base(srcRel)
	dstAbs := filepath.Join(s.root, filepath.FromSlash(dstGroupPath), srcBase)
	// 2026-06-29 增:no-op 短路。dstAbs == srcAbs 表示目标位置就是当前位置,
	// 用户操作"挪到根"(顶层分组)或"挪到自己父级下"都会落到这里。
	// 走 os.Rename 也能 noop,但前端在 store 层前就会先撞到"目标已存在"判
	// 断导致误报,所以这里先返回 OK。
	if dstAbs == srcAbs {
		return nil
	}
	// 2026-06-29 增:防御性 ancestor check。用 group path 判(src=aa/yy → noop
	// 时 dstAbs=aa/yy,等于 src;src=aa/yy → dst=aa/zz 时 dstAbs=aa/zz/yy,在
	// src 外;src=aa → dst=aa/yy 时 dstAbs=aa/yy/aa,在 src 内 — 这才是真正
	// 会死循环的情况)。
	// 用 isDescendantOrSame 判 abs 关系能精准捕获"src 在 dst 内部"或"dst 在
	// src 内部",但挪到根(src=aa,dst=""→dstAbs=root/aa=srcAbs)会被误判为
	// "挪到自己"。所以挪到根特例先放行。
	// (root 这个特例也走 copyDirRecursive 兜底,真出问题也会被拦下)
	if dstGroupPath != "" && isDescendantOrSame(dstAbs, srcAbs) {
		return fmt.Errorf("skillstore: cannot move group %q into its own descendant %q", srcGroupPath, dstGroupPath)
	}
	// 目标已存在 → 拒覆盖
	if _, err := os.Stat(dstAbs); err == nil {
		return fmt.Errorf("skillstore: target group %q already exists", dstAbs)
	}
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return err
	}
	if err := os.Rename(srcAbs, dstAbs); err != nil {
		if cerr := copyDirRecursive(srcAbs, dstAbs); cerr != nil {
			return fmt.Errorf("skillstore: move group failed (rename=%v, copy=%v)", err, cerr)
		}
		if rerr := os.RemoveAll(srcAbs); rerr != nil {
			return fmt.Errorf("skillstore: move group source cleanup failed: %w", rerr)
		}
	}
	_ = removeIfEmpty(filepath.Dir(srcAbs))
	return nil
}

// CreateGroupDir 创建分组目录(groupPath 可多级,如 "frontend/react")。
// 已存在不报错(幂等)。
//
// 2026-06-29 增:供 cskill.create_group 用。
func (s *Store) CreateGroupDir(groupPath string) error {
	if groupPath == "" {
		return nil
	}
	rel, err := safeRelPath(groupPath)
	if err != nil {
		return fmt.Errorf("skillstore: invalid group path %q: %w", groupPath, err)
	}
	abs := filepath.Join(s.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return fmt.Errorf("skillstore: mkdir group %s: %w", abs, err)
	}
	return nil
}

// DeleteGroupDir 删分组目录及其子树。groupPath 为空时返回 nil。
// recursive=false 时,若分组非空,返回 (deleted_paths, error)(不递归删子项,
// 让 caller 决定是否强删)。
//
// 2026-06-29 增:供 cskill.delete_group 用。
// deleted 数组是"该分组下所有 skill 叶子的相对路径"(供前端在 cascade=true 时
// 同步工具目录),即使删除失败也尽量填好让 caller 做部分回滚。
func (s *Store) DeleteGroupDir(groupPath string, recursive bool) ([]string, error) {
	if groupPath == "" {
		return nil, nil
	}
	rel, err := safeRelPath(groupPath)
	if err != nil {
		return nil, fmt.Errorf("skillstore: invalid group path %q: %w", groupPath, err)
	}
	abs := filepath.Join(s.root, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skillstore: group path %s is not a dir", abs)
	}
	var deleted []string
	s.collectSkillLeafPaths(abs, rel, &deleted)
	if !recursive && len(deleted) > 0 {
		return deleted, fmt.Errorf("skillstore: group %s is not empty (contains %d skills)", groupPath, len(deleted))
	}
	if err := os.RemoveAll(abs); err != nil {
		return deleted, fmt.Errorf("skillstore: remove group %s: %w", abs, err)
	}
	_ = removeIfEmpty(filepath.Dir(abs))
	return deleted, nil
}

// RenameGroupDir 重命名分组的最后一段(不挪父级,父路径保持不变)。
// srcGroupPath 可多级(如 "frontend/react"),newName 是单段名(不含 '/')。
// 同层同名目录已存在 → 返回 error(避免覆盖)。newName 与旧 base 相同 → 幂等返回 nil。
//
// 2026-06-29 增:为支持"分组右键重命名"。
// 实现策略:整个目录用 os.Rename,跨设备降级 copy+delete(同 MoveGroupPath)。
func (s *Store) RenameGroupDir(srcGroupPath string, newName string) (string, error) {
	if srcGroupPath == "" {
		return "", fmt.Errorf("skillstore: rename group: empty src group path")
	}
	if newName == "" || strings.ContainsAny(newName, "/\\") {
		return "", fmt.Errorf("skillstore: rename group: invalid new name %q (must be a single segment)", newName)
	}
	srcRel, err := safeRelPath(srcGroupPath)
	if err != nil {
		return "", fmt.Errorf("skillstore: rename group: bad src path %q: %w", srcGroupPath, err)
	}
	srcAbs := filepath.Join(s.root, filepath.FromSlash(srcRel))
	if _, err := os.Stat(srcAbs); err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	srcBase := filepath.Base(srcRel)
	if srcBase == newName {
		// 名字未变 → 幂等返回
		return srcRel, nil
	}
	dstAbs := filepath.Join(filepath.Dir(srcAbs), newName)
	if _, err := os.Stat(dstAbs); err == nil {
		return "", fmt.Errorf("skillstore: rename group: target %q already exists", newName)
	}
	if err := os.Rename(srcAbs, dstAbs); err != nil {
		if cerr := copyDirRecursive(srcAbs, dstAbs); cerr != nil {
			return "", fmt.Errorf("skillstore: rename group failed (rename=%v, copy=%v)", err, cerr)
		}
		if rerr := os.RemoveAll(srcAbs); rerr != nil {
			return "", fmt.Errorf("skillstore: rename group: source cleanup failed: %w", rerr)
		}
	}
	// 返回新相对路径(用 '/' 分隔,前端直接消费)
	newRel, _ := filepath.Rel(s.root, dstAbs)
	return filepath.ToSlash(newRel), nil
}

// collectSkillLeafPaths 递归收集 group abs 目录下的所有 skill 叶子路径(相对 root),
// 结果用 '/' 分隔,append 到 out。
func (s *Store) collectSkillLeafPaths(abs, relGroup string, out *[]string) {
	if _, err := os.Stat(filepath.Join(abs, "SKILL.md")); err == nil {
		*out = append(*out, filepath.ToSlash(filepath.Join(relGroup, filepath.Base(abs))))
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s.collectSkillLeafPaths(filepath.Join(abs, e.Name()), filepath.ToSlash(filepath.Join(relGroup, e.Name())), out)
	}
}

// copyDirRecursive 递归复制 src 目录到 dst(覆盖式);用于跨设备 MoveGroupPath 兜底。
//
// 2026-06-29 增:加防御性 ancestor check — 如果 dst 在 src 内部(含 dst == src),
// 立即返回 error。原因是 caller(MoveGroupPath / MoveGroupDir)若没拦住
// "把目录挪到自己子目录" 的情况,os.MkdirAll(dst) 会在 src 内创建一个新子目录,
// 然后 ReadDir(src) 会扫到这个新子目录,递归 copy,死循环直到 macOS
// 路径长度 255 字节上限才崩(tmp 下出现几百层 yy/aa/yy/aa/...)。
// 失败路径在 caller 侧(MoveGroupPath / MoveGroupDir)的 normalize 之前
// 就该被拦下,这里只是兜底,确保 copyDirRecursive 永不进入这种状态。
func copyDirRecursive(src, dst string) error {
	if isDescendantOrSame(dst, src) {
		return fmt.Errorf("copyDirRecursive: dst %q is inside src %q (refusing to recurse)", dst, src)
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("copyDirRecursive: %s is not a dir", src)
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDirRecursive(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := writeFileAtomic(dstPath, string(content), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// List 列出全部 skill 的 Canonical(目录扫描 + frontmatter 解析)。
// 损坏的 skill 跳过,不阻塞整体;keyword 非空时做 name 子串匹配(不区分大小写)。
//
// 2026-06-29 改:支持分组子目录 — 递归扫 root(深度 maxScanDepth,继承自
// skilladapter.BaseAdapter 的常量),叶子 = 有 SKILL.md 的目录。返回的每个
// Canonical.Manifest.GroupPath 都已自动回填(由目录相对 root 的路径反推),
// Manifest.Name 是叶子目录名。
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
		// 顶层入口:每个 entry 既可能是 skill 叶子,也可能是分组目录
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			// 隐藏目录视为非 skill(避免 .system / .curated 这类系统子目录混入)
			continue
		}
		// 用 walkSkills 风格的递归:遇到 SKILL.md 即停止,否则继续下钻
		s.collectSkillsRecursive(filepath.Join(s.root, name), "", kw, 0, &out)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Manifest.GroupPath != out[j].Manifest.GroupPath {
			return out[i].Manifest.GroupPath < out[j].Manifest.GroupPath
		}
		return out[i].Manifest.Name < out[j].Manifest.Name
	})
	return out, nil
}

// collectSkillsRecursive 递归找叶子 skill 目录。
//
// 设计要点(2026-06-29):与 skilladapter.BaseAdapter.Scan 类似,但去掉
// system-path skip(库内不区分 system / user)+ 去掉 LocalName normalize
// (库内的 name 已经是规约过的叶子名)。
//
// 参数 groupPath 是当前递归层级相对 root 的路径(用 '/' 分隔),
// 用于回填 Manifest.GroupPath。
func (s *Store) collectSkillsRecursive(absDir, groupPath string, kw string, depth int, out *[]skilladapter.Canonical) {
	if depth > maxScanDepth {
		return
	}
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return
	}
	// 自身有 SKILL.md → 视为 skill 叶子,停止下钻
	if _, err := os.Stat(filepath.Join(absDir, "SKILL.md")); err == nil {
		c, err := s.loadFromDir(absDir)
		if err != nil {
			return // 损坏的 skill 跳过
		}
		if kw != "" && !strings.Contains(strings.ToLower(c.Manifest.Name), kw) {
			return
		}
		*out = append(*out, *c)
		return
	}
	// 否则继续下钻(分组中间层)
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !e.IsDir() {
			// 跳过文件,只看目录
			continue
		}
		childAbs := filepath.Join(absDir, name)
		childGroup := name
		if groupPath != "" {
			childGroup = groupPath + "/" + name
		}
		s.collectSkillsRecursive(childAbs, childGroup, kw, depth+1, out)
	}
}

// TreeNode 树形节点,供 ListTree 返回。Group 节点 = 中间目录;Skill 节点 = 叶子 skill。
//
// 2026-06-29 增:JSON tag 用 snake_case,便于前端直接消费。
type TreeNode struct {
	// Name 是节点名(不含父路径;Skill = 叶子 name;Group = 该段目录名)
	Name string `json:"name"`
	// Path 是节点相对 root 的完整路径(Group = "frontend/react";Skill = "frontend/react/use-cache")
	Path string `json:"path"`
	// IsGroup 区分是分组还是 skill;true = 分组(可能含子树),false = skill 叶子
	IsGroup bool `json:"is_group"`
	// Children 仅 IsGroup=true 时有效;按字典序排序(Skill 排在 Group 后面或混排都可,
	// 前端可按需重排)。叶子 skill 时为空数组。
	Children []TreeNode `json:"children"`
	// SkillMeta 仅 IsGroup=false 时有效;包含 skill 的轻量元数据
	// (前端列表项展示用,避免再发一次 list 请求)。
	SkillMeta *SkillTreeMeta `json:"skill_meta,omitempty"`
}

// SkillTreeMeta 树节点中携带的 skill 轻量元数据。
//
// 2026-06-29 增:AppliedTools 是该 skill 被全局启用的工具 ID 列表(从
// cskillapply 的 scope-status 反推),供前端卡片"被这些工具全局调用了"显示。
// 复用了 cskill 包里的 GlobalAppliedTools helper(同进程),避免在 store 层
// 重复实现 scope-status 扫描逻辑。
type SkillTreeMeta struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Triggers     []string `json:"triggers"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
	AppliedTools []string `json:"applied_tools,omitempty"`
}

// ListTree 列出全部 skill 的树形结构(供前端分组 UI 用)。
//
// 2026-06-29 增:返回嵌套 TreeNode 数组,root 节点的 IsGroup=true + Children 列出
// 顶层项;keyword 非空时,对 skill 叶子做 name 子串匹配(分组即使不含匹配项也保留,
// 便于前端展示"匹配项所在的分组链")。
func (s *Store) ListTree(keyword string) ([]TreeNode, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	var roots []TreeNode
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		node := s.buildTreeNode(filepath.Join(s.root, name), name, "", kw, 0)
		if node == nil {
			continue
		}
		roots = append(roots, *node)
	}
	sortTreeNodes(roots)
	return roots, nil
}

// buildTreeNode 递归构造 TreeNode。
//
// 返回 nil 表示该子树在 keyword 过滤后无任何匹配,前端可以隐藏。
// 否则:
//   - 若自身是 skill 叶子 → IsGroup=false,Children=[]
//   - 否则 IsGroup=true,Children 含子树
func (s *Store) buildTreeNode(absDir, name, groupPath, kw string, depth int) *TreeNode {
	if depth > maxScanDepth {
		return nil
	}
	// 自身是 skill 叶子
	if _, err := os.Stat(filepath.Join(absDir, "SKILL.md")); err == nil {
		c, err := s.loadFromDir(absDir)
		if err != nil {
			return nil
		}
		// keyword 过滤:不匹配直接丢掉(分组会因而被折叠)
		if kw != "" && !strings.Contains(strings.ToLower(c.Manifest.Name), kw) {
			return nil
		}
		fi := dirModTime(absDir)
		return &TreeNode{
			Name:    c.Manifest.Name,
			Path:    joinGroupPath(groupPath, c.Manifest.Name),
			IsGroup: false,
			SkillMeta: &SkillTreeMeta{
				Name:        c.Manifest.Name,
				Version:     c.Manifest.Version,
				Description: c.Manifest.Description,
				Triggers:    c.Manifest.Triggers,
				UpdatedAt:   fi,
			},
		}
	}
	// 否则是分组中间层:递归收集子树
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil
	}
	var children []TreeNode
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		childAbs := filepath.Join(absDir, e.Name())
		childGroup := joinGroupPath(groupPath, name)
		child := s.buildTreeNode(childAbs, e.Name(), childGroup, kw, depth+1)
		if child == nil {
			continue
		}
		children = append(children, *child)
	}
	if len(children) == 0 {
		// 空分组:如果 keyword 为空(默认列出全部)就保留,让用户能看到空目录;否则隐藏
		if kw != "" {
			return nil
		}
	}
	return &TreeNode{
		Name:     name,
		Path:     joinGroupPath(groupPath, name),
		IsGroup:  true,
		Children: children,
	}
}

// joinGroupPath 安全拼接分组路径(空段跳过)。
func joinGroupPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "/" + child
}

// sortTreeNodes 对树节点按 (IsGroup desc, Name asc) 排序 — 分组在前,叶子在后,
// 各自按字典序。
func sortTreeNodes(nodes []TreeNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].IsGroup != nodes[j].IsGroup {
			return nodes[i].IsGroup
		}
		return nodes[i].Name < nodes[j].Name
	})
	for i := range nodes {
		if nodes[i].IsGroup && len(nodes[i].Children) > 0 {
			sortTreeNodes(nodes[i].Children)
		}
	}
}

// --- internals ---

// skillDir 计算某个 skill 的实际目录(无 groupPath,等价于"根下直接子目录")。
//
// 2026-06-29 改:为支持多级分组,旧 API 走"根下直接子目录"的语义保持不变;
// 新代码请用 resolveSkillDir(groupPath, name) 取分组路径下的目录。
func (s *Store) skillDir(name string) (string, error) {
	return s.resolveSkillDir("", name)
}

// resolveSkillDir 把 (groupPath, name) 解析到 root 下的绝对目录,支持多级分组。
//
// groupPath 允许 '/',内部走 safeRelPath 防穿越;name 仍走 NormalizeName 规约
// (不含 '/')。返回绝对路径,出错返回 (零值, error)。
func (s *Store) resolveSkillDir(groupPath string, name string) (string, error) {
	rel := name
	if groupPath != "" {
		rel = filepath.ToSlash(filepath.Join(groupPath, name))
	}
	cleaned, err := safeRelPath(rel)
	if err != nil {
		return "", fmt.Errorf("skillstore: invalid skill path %q: %w", rel, err)
	}
	return filepath.Join(s.root, filepath.FromSlash(cleaned)), nil
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

// dirModTime 读 dir 下 SKILL.md 的 mtime(给 list 提供"最近修改"字段)。
// 不可读时返回空串(原 fileModTime 在 sskill 包是同名同语义,这里 store 内
// 自带一份避免反向依赖)。
func dirModTime(dir string) string {
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return ""
	}
	return info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
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

// isDescendantOrSame 2026-06-29 增:判断 dst 是否在 src 内部,或 dst == src。
// 用途:copyDirRecursive / MoveGroupPath / MoveGroupDir 拦截"把目录挪到自己子目录"
// 的非法操作(否则会进入死循环:os.MkdirAll 在 src 内创建 dst,然后 ReadDir(src)
// 扫到 dst,递归,死循环直到路径长度超限)。
//
// 实现:把两边 Clean 之后,如果 dst == src 或 dst 是 src 父目录的子路径,返回 true。
// 跨平台:用 filepath.Clean 走 OS 路径分隔符。
func isDescendantOrSame(dst, src string) bool {
	cleanDst := filepath.Clean(dst)
	cleanSrc := filepath.Clean(src)
	if cleanDst == cleanSrc {
		return true
	}
	// rel, _ := filepath.Rel(cleanSrc, cleanDst):dst 相对 src 的路径
	// 若以 .. 开头 → dst 在 src 外(不是后代);否则就是后代或自身
	rel, err := filepath.Rel(cleanSrc, cleanDst)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	// 以 .. 开头 = 跳出 src 的子树;否则(没有 .. 前缀)就是 src 的子路径
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
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
