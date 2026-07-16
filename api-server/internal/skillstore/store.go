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
	"unicode/utf8"

	"ginp-api/configs"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillversion"
	sharefunc "ginp-api/share/func"
)

// ErrNotFound skill 不存在。
var ErrNotFound = errors.New("skillstore: not found")

// 2026-07-05 增:ErrCorruptedFile 表示磁盘文件被破坏(含非 UTF-8 字节)。
// 前端可以字符串匹配 "non-UTF-8" / "已损坏" 来弹清晰提示(比通用 500 更友好)。
var ErrCorruptedFile = errors.New("skillstore: file contains non-UTF-8 bytes")

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
//
// 2026-07-12 改:deletedPaths 是前端"明确删除"路径列表(相对 skill 根的 rel path),
// WalkDir 复制阶段命中即跳过 — 让外层 RemoveAll(dir) 真正物理删除这些路径,
// 避免 "Save 复活了前端已删除的目录" 的 bug。传 nil/空等价于旧版行为(完全向后兼容)。
func (s *Store) Save(c skilladapter.Canonical, deletedPaths []string) error {
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
		// 2026-07-12 改:业务占位 .skillbox-placeholder 转 mkdir — 不写文件,
		// 只确保父目录存在。这是"空目录标识"的真正语义:占位条目代表
		// <dir>/ 是一个空目录,MkdirAll(filepath.Dir(dst)) 后 dst 这个
		// 占位文件路径本身就是这个空目录。上一版的"continue"会把空目录
		// 吞掉,导致前端任何 updateSkill(重命名 / 编辑文件)后磁盘上空
		// 目录永久消失。
		if filepath.Base(rel) == ".skillbox-placeholder" {
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return fmt.Errorf("skillstore: mkdir %s: %w", filepath.Dir(dst), err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("skillstore: mkdir %s: %w", filepath.Dir(dst), err)
		}
		if err := writeFileAtomic(dst, f.Content, 0o644); err != nil {
			return err
		}
	}

	// 原子替换:先把目标目录(如果存在)删了,再 rename temp -> target。
	//
	// 2026-07-12 改:删旧目录前先扫一遍,如果里面有 .skillbox-placeholder 残留
	// (旧版本 Save 写进去的),递归删掉 — 这些是空目录占位条目,正常路径不该
	// 出现在磁盘上。否则删除 skill 时用户也会看到目录里残留 .skillbox-placeholder。
	if _, statErr := os.Stat(dir); statErr == nil {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Base(p) == ".skillbox-placeholder" {
				_ = os.Remove(p)
			}
			return nil
		})
	}
	// 2026-07-12 改:不再 RemoveAll(dir) 全量覆盖 — 这是空目录消失的根因。
	// 旧版 Save 是"覆盖式": tmp 重建完整目录树 → 删旧目录 → rename。
	// 问题:前端 state 未必包含磁盘上所有空目录的占位条目(比如前端只 push
	// 了 dirty 的 SKILL.md,没遍历 state 包含占位),Save 后磁盘上空目录
	// 全部丢失 — 用户报告"重命名 cc→dd 后 cc/dd 都消失了"。
	//
	// 新版: tmp 已经按 c.Files 重建了完整目录树(包括占位目录)。
	// 此时把"磁盘原 dir 里 tmp 没有的非占位文件"复制到 tmp(保留磁盘上
	// 前端不知道的文件/空目录树),然后再 RemoveAll + rename 即可。
	//
	// 关键不变量:
	//   - 用户文件(SKILL.md / 普通 .md / .json 等):c.Files 里有的覆盖、tmp
	//     没有的保留(从原 dir 复制过去)。
	//   - 空目录:c.Files 里有占位条目 → tmp 已有该目录;c.Files 里没有但磁盘
	//     上有空目录 → 复制过去保留(目录本身会被 WalkDir 自动处理)。
	//   - .skillbox-placeholder:绝不能复制到 tmp(避免重新引入占位文件)。
	if _, statErr := os.Stat(dir); statErr == nil {
		_ = filepath.WalkDir(dir, func(srcPath string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if srcPath == dir {
				return nil
			}
			rel, relErr := filepath.Rel(dir, srcPath)
			if relErr != nil {
				return nil
			}
			// 2026-07-12 增:命中 deletedPaths(精确匹配 / prefix 子树匹配)→ 跳过复制,
			// 让外层 RemoveAll(dir) 真正物理删除该路径。前端 "删文件夹/删文件"
			// 会把这个列表带过来;其它入口(普通保存 / 重命名 / 导入)传 nil,行为不变。
			if isDeletedPath(rel, deletedPaths) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// 跳过占位文件本身
			if !d.IsDir() && filepath.Base(srcPath) == ".skillbox-placeholder" {
				return nil
			}
			dstPath := filepath.Join(tmp, rel)
			if _, existsErr := os.Stat(dstPath); existsErr == nil {
				// tmp 已有这个路径(来自 c.Files 的某个 file/占位目录),不覆盖
				return nil
			}
			// tmp 缺这个路径,从 src 复制到 dst(保留磁盘上前端不知道的内容)
			if d.IsDir() {
				if err := os.MkdirAll(dstPath, 0o755); err != nil {
					return nil
				}
				return nil
			}
			if err := copyFileAtomic(srcPath, dstPath); err != nil {
				return nil
			}
			return nil
		})
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("skillstore: remove old dir: %w", err)
	}
	if err := os.Rename(tmp, dir); err != nil {
		return fmt.Errorf("skillstore: rename temp: %w", err)
	}

	// 2026-07-17 增:落盘成功后调 skillversion.AutoCommitAndPush。
	// 失败仅写 logger,不阻断 store.Save(业务写盘已经成功,版本管理失败不能反向回滚)。
	// 走 goroutine 异步执行,store.Save 不等 git 完成。
	go autoCommitAfterSave(c.Manifest.GroupPath, c.Manifest.Name, "update")
	return nil
}

// autoCommitAfterSave 把 store.Save 的成功事件转成 git commit。
//
// 2026-07-17:走 skillversion.Repo.AutoCommitAndPush,失败不抛(已包在内部)。
// 注释:go-git 调用方在异步 goroutine 写日志,要复用项目 logger。
func autoCommitAfterSave(group, name, op string) {
	defer func() {
		_ = recover() // 防 panic 拖垮 store 调用方
	}()
	rel := filepath.ToSlash(filepath.Join(group, name))
	if group == "" {
		rel = name
	}
	repo, err := skillversionRepo()
	if err != nil {
		loggerWarn("skillversion: open repo: %v", err)
		return
	}
	_, err = repo.AutoCommitAndPush(skillversion.CommitInput{
		Message: fmt.Sprintf("skill(store): %s %s", op, rel),
		Paths:   []string{rel},
	})
	if err != nil {
		loggerWarn("skillversion: AutoCommitAndPush %s: %v", rel, err)
	}
}

// skillversionRepo 拿 skillversion 全局单例(避免每次 Save 都重新 init)。
var skillversionRepo = func() (*skillversion.Repo, error) {
	return skillversion.Default()
}

// loggerWarn 占位 logger,替换为 stderr 输出(skillversion 失败仅用于调试)。
var loggerWarn = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[skillversion] "+format+"\n", args...)
}

// copyFileAtomic 把 src 单文件复制到 dst(读 src 内容 → writeFileAtomic dst)。
// 用于 Save 阶段"保留磁盘上前端不知道的文件"场景。
func copyFileAtomic(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return writeFileAtomic(dst, string(content), 0o644)
}

// isDeletedPath 判断 rel 是否命中 deletedPaths 任一项。
//
// 2026-07-12 增:精确匹配 → 命中;prefix 子树匹配(deleted + "/") → 命中整棵子树;
// 空字符串 / nil 列表 → 一律 false(等价于旧版"不删任何东西",行为完全兼容)。
func isDeletedPath(rel string, deleted []string) bool {
	if len(deleted) == 0 {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, p := range deleted {
		if p == "" {
			continue
		}
		p = filepath.ToSlash(strings.TrimRight(p, "/"))
		if rel == p {
			return true
		}
		if strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
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
	// 2026-07-05 改:校验 SKILL.md 是不是合法 UTF-8。磁盘文件被破坏
	// (如 Finder 拖拽 / iCloud sync 把 plist 二进制混进文本)时,直接 string(content)
	// 会把非法字节静默替换成 U+FFFD,前端渲染出豆腐块。这里检测到非 UTF-8
	// 时返回 sentinel ErrCorruptedFile,前端能用 errors.Is 识别并弹清晰提示。
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("%w: %s", ErrCorruptedFile, skillMD)
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
	// 2026-07-11 增:补全空目录 — walkFiles 只返回有文件的目录里的文件,
	// 磁盘上"空目录"(如 dd/)在 files 数组里没有对应条目,前端 buildTree
	// 拿不到任何 <dir>/ 子文件,空目录永远不显示。这里给每个空目录补一个
	// .skillbox-placeholder 占位条目(后端 store.Save 看到也会 mkdir 真实
	// 占位文件,前端 FileTreeView.buildTree 走 BUSINESS_PLACEHOLDERS 白名单
	// 知道它是占位)。
	emptyDirs, ederr := listEmptyDirs(dir)
	if ederr == nil {
		for _, d := range emptyDirs {
			c.Files = append(c.Files, skilladapter.File{
				Path:    filepath.ToSlash(filepath.Join(d, ".skillbox-placeholder")),
				Content: "",
			})
		}
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
	// 把"技能磁盘根"写到 SourceDir,与 skilladapter.BaseAdapter.readSkillDir
	// 保持一致:用 EvalSymlinks 解析真实路径(避免 symlink 链上 path 在 ~/.claude/skills
	// 与 ~/.agents/skills/xxx 之间漂移)。home 端从 store load 的 skill 走
	// ApplyLink 时必须依赖 SourceDir(SourceDir == "" 会触发 "empty source_dir")。
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		c.SourceDir = real
	} else {
		c.SourceDir = dir
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

// findByName 在 store 树里按叶子 name(目录名 == skill name)找到第一条匹配,
// 返回其绝对目录路径和 groupPath(供 LoadByPath / ExistsByPath / DeleteByPath
// 复用)。找不到返 ("", "", ErrNotFound)。
//
// 2026-07-03 增:为兼容旧 API 语义(只按 name 定位,不要求 caller 给 groupPath),
// 在多级分组下也能找到 aa/debug-helper 这种嵌套 skill。设计要点:
//   - 浅层优先:根下的 skill 优先于分组里的同名 skill(便于后续真出现重名时,
//     行为可预测 — 旧 store 没有多级时就是只走根)。
//   - 唯一性:全树扫描发现多个同名 → 仍返浅层第一个(根 → aa → aa/bb → …),
//     避免静默歧义。
func (s *Store) findByName(name string) (absDir, groupPath string, err error) {
	if strings.TrimSpace(name) == "" {
		return "", "", ErrNotFound
	}
	// 浅层优先:用 BFS 走 root → 第一层子目录 → 第二层 → …,碰到 name 即停。
	type qitem struct {
		absDir    string
		groupPath string
	}
	queue := []qitem{{absDir: s.root, groupPath: ""}}
	for len(queue) > 0 {
		head := queue[0]
		queue = queue[1:]
		entries, err := os.ReadDir(head.absDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			en := e.Name()
			if strings.HasPrefix(en, ".") {
				continue
			}
			child := filepath.Join(head.absDir, en)
			// 自身有 SKILL.md → 视为叶子,目录名 == skill name 才算命中
			if _, err := os.Stat(filepath.Join(child, "SKILL.md")); err == nil {
				if en == name {
					rel, rerr := filepath.Rel(s.root, child)
					if rerr == nil {
						rel = filepath.ToSlash(rel)
						gp := ""
						if rel != "." && rel != "" {
							gp = filepath.Dir(rel)
							if gp == "." {
								gp = ""
							}
						}
						return child, gp, nil
					}
				}
				continue
			}
			// 否则是分组中间层,加入队列继续 BFS
			childGP := en
			if head.groupPath != "" {
				childGP = head.groupPath + "/" + en
			}
			queue = append(queue, qitem{absDir: child, groupPath: childGP})
		}
	}
	return "", "", ErrNotFound
}

// LoadByName 在全树按 name 找 skill,自动解析 groupPath 后 Load。
// 找不到返 ErrNotFound。多级分组改造后,旧 API(只给 name)需要这层桥接。
//
// 2026-07-03 增。
func (s *Store) LoadByName(name string) (*skilladapter.Canonical, error) {
	dir, _, err := s.findByName(name)
	if err != nil {
		return nil, err
	}
	return s.loadFromDir(dir)
}

// ExistsByName 全树按 name 判断 skill 是否存在。
//
// 2026-07-03 增。
func (s *Store) ExistsByName(name string) bool {
	dir, _, err := s.findByName(name)
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

// DeleteByName 全树按 name 找 skill 并物理删除(整目录)。找不到返 nil(幂等)。
//
// 2026-07-03 增。
func (s *Store) DeleteByName(name string) error {
	dir, _, err := s.findByName(name)
	if err != nil {
		return nil // 幂等
	}
	return s.deleteDir(dir)
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
	// 2026-07-08 改:不再清理空 group 目录。
	// 原逻辑在 451-452 行调 removeIfEmpty(srcParent),把"移走最后一个 skill 后
	// 变空的源 group 目录"删掉,导致首页树把空 group 隐藏。
	// 用户原话"分组 a 移空后也要显示",所以保留空 group 目录,
	// 删分组需要走 DeleteGroupDir 显式操作。
	return nil
}

// RenameSkillInGroup 2026-07-11 增:把 skill 目录在同分组内换名(只改最后一段,groupPath 不变)。
// 与 MoveGroupPath 区别:dst 仍走 srcGroupPath,只换 name;同时锁定 src + dst 防
// 并发覆盖。校验 newName 非空 + 与 oldName 不一致 + 目标不存在。
// 成功返回新相对路径(groupPath/newName,'/' 分隔,前端直接消费)。
func (s *Store) RenameSkillInGroup(srcGroupPath string, oldName string, newName string) (string, error) {
	if oldName == "" {
		return "", fmt.Errorf("skillstore: rename skill: old name is empty")
	}
	if newName == "" {
		return "", fmt.Errorf("skillstore: rename skill: new name is empty")
	}
	if oldName == newName {
		return "", fmt.Errorf("skillstore: rename skill: old and new name are the same")
	}
	// groupPath 允许空(根),但含 .. / 绝对路径 / NUL 仍拒。safeRelPath 在空串
	// 时直接报"empty path",这里把空串视作合法,让"根下 skill"场景能走通。
	cleanGroup := ""
	if srcGroupPath != "" {
		cp, err := safeRelPath(srcGroupPath)
		if err != nil {
			return "", fmt.Errorf("skillstore: rename skill: bad group path %q: %w", srcGroupPath, err)
		}
		cleanGroup = cp
	}
	srcAbs, err := s.resolveSkillDir(cleanGroup, oldName)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(srcAbs, "SKILL.md")); err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	dstAbs, err := s.resolveSkillDir(cleanGroup, newName)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(dstAbs); err == nil {
		return "", fmt.Errorf("skillstore: rename skill: target %q already exists", newName)
	}
	// 跨目录并发的 file lock — 锁定 src(防止别人正在写)+ 锁 dst(防止同名创建竞态)。
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return "", fmt.Errorf("skillstore: rename skill: mkdir parent: %w", err)
	}
	srcUnlock, err := s.lockScope(srcAbs)
	if err != nil {
		return "", err
	}
	defer srcUnlock()
	dstUnlock, err := s.lockScope(dstAbs)
	if err != nil {
		return "", err
	}
	defer dstUnlock()
	// 再确认一次目标不存在(并发创建场景)
	if _, err := os.Stat(dstAbs); err == nil {
		return "", fmt.Errorf("skillstore: rename skill: target %q already exists", newName)
	}
	if err := os.Rename(srcAbs, dstAbs); err != nil {
		// 跨设备 / 异常 → copy + remove
		if cerr := copyDirRecursive(srcAbs, dstAbs); cerr != nil {
			return "", fmt.Errorf("skillstore: rename skill failed (rename=%v, copy=%v)", err, cerr)
		}
		if rerr := os.RemoveAll(srcAbs); rerr != nil {
			return "", fmt.Errorf("skillstore: rename skill: source cleanup failed: %w", rerr)
		}
	}
	// 返回新相对路径(groupPath/newName,'/' 分隔)
	newRel := newName
	if cleanGroup != "" && cleanGroup != "." {
		newRel = filepath.ToSlash(filepath.Join(cleanGroup, newName))
	}
	return newRel, nil
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
			// 2026-07-05 改:损坏的 skill 跳过时 log 一条 warn,
			// 用户可以在 ~/.skill-box/logs 里看到具体哪个目录的 SKILL.md 坏了。
			fmt.Fprintf(os.Stderr, "[skillstore] skip corrupted skill %s: %v\n", absDir, err)
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
// 2026-07-12 增:SourcePath 是该 skill 的"导入来源磁盘绝对路径"(EvalSymlinks
// 后的真实路径)。仅在 skill 是从外部位置(如 ~/.agents/skills/)导入到 store
// 时才会有值;前端据此判断"全局 Agent"标签。落盘位置:skill 目录下的
// .skillbox-source.json sidecar 文件(避免污染 SKILL.md frontmatter)。
type SkillTreeMeta struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Triggers     []string `json:"triggers"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
	AppliedTools []string `json:"applied_tools,omitempty"`
	// 2026-07-12 增:导入来源磁盘绝对路径(可选)。前端用 [\\/]\.agents[\\/]skills[\\/]
	// 正则判断是否来自 ~/.agents/skills/ 全局 Agent 目录,贴"全局 Agent"标签。
	SourcePath   string   `json:"source_path,omitempty"`
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
		// 2026-07-12 改:实时检测该 skill 是否在 ~/.agents/skills/ 全局目录下,
		// 而不是依赖 sidecar 缓存。理由:全局目录是用户家目录的共享 skills 池,
		// skillbox 应当"按需"反映其状态(用户在 ~/.agents/skills/ 增删 skill
		// 后,下次 reload 列表立即生效),而非用 sidecar 记录"曾经导入过"
		// 这种历史状态。命中条件:磁盘上存在 ~/.agents/skills/<name>/SKILL.md。
		// 跨平台 home:用 os.UserHomeDir(同 onboarding 包的 osUserHomeDir)。
		srcPath := resolveGlobalSourcePath(c.Manifest.Name)
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
				SourcePath:  srcPath,
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
		// 2026-07-07 改:过滤掉 macOS 系统元数据文件(.DS_Store / ._*) 和其他隐藏文件。
		// 旧版 walkFiles 不过滤,.DS_Store 二进制被原样读进 Content → 序列化为 JSON 时含
		// 大量非法 \uXXXX escape 序列 → 前端 JSON.parse 抛 SyntaxError: Unexpected token。
		// 附带副作用:响应体从几 KB 涨到几百 KB(.DS_Store 二进制 → UTF-8 转义后巨大)。
		// 过滤规则跟前端 FileTreeView.buildTree 保持一致:
		//   - rel 任一段以 . 开头 → 跳过
		//   - 顶层或子目录里的隐藏文件都不保留
		if rel == "" || rel == "." {
			return nil
		}
		segs := strings.Split(filepath.ToSlash(rel), "/")
		for _, seg := range segs {
			if strings.HasPrefix(seg, ".") {
				return nil
			}
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

// 2026-07-11 增:扫目录收集所有"空目录"(递归)。
// 用于 loadFromDir 把磁盘上空目录也展示在文件树里 — 不然前端 buildTree
// 拿不到任何 <dir>/ 子条目,空目录永远不显示。
// 过滤规则跟 walkFiles 一致:任一段以 . 开头的目录跳过(.git / .DS_Store 等)。
func listEmptyDirs(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// 跳过根本身
		if path == root {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		relSlash := filepath.ToSlash(rel)
		// 过滤隐藏目录
		for _, seg := range strings.Split(relSlash, "/") {
			if strings.HasPrefix(seg, ".") {
				// 跳过整个子树
				return filepath.SkipDir
			}
		}
		// 看这个目录是不是"空"的(没有任何 entry — 既没文件也没子目录)
		// WalkDir 已扫过这个 dir,直接 os.ReadDir 看 entries
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return readErr
		}
		if len(entries) == 0 {
			out = append(out, relSlash)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
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

// ============================================
// 2026-07-12 增:全局 Agent 实时检测
// ============================================
//
// 设计动机:首页 skill 卡片要显示"全局 Agent"标签,语义是"该 skill
// 当前在 ~/.agents/skills/ 全局 Agent 目录下"。store 不缓存"曾经
// 导入过"这种历史状态 — 每次 ListTree 都实时检查磁盘:
//   1. 拼出候选路径 ~/.agents/skills/<name>/SKILL.md
//   2. EvalSymlinks 拿真实路径(macOS /private/var/...)
//   3. 存在则把该路径作为 SourcePath 注入,前端用正则判定后贴 tag
//
// 这样做的好处:
//   - 用户在 ~/.agents/skills/ 下增 / 删 skill 后,下次 reload 列表立即生效
//   - 不需要任何 sidecar 缓存文件,store 保持"只读系统"的纯粹性
//   - 不依赖"曾经导入过"的历史信息,跟磁盘真值同步
//
// 跟 onboarding 包的 resolveGlobalSkillsRoot 区别:
//   - onboarding 包遍历 skilladapter.All() 的 DiscoverPaths 拿到所有候选路径
//     (claude/codex/cline 等多 adapter 都有 ~/.agents/skills,会去重)
//   - 这里只关心"该 skill name 在 .agents/skills/<name> 下是否存在",
//     简单 stat 即可,不需要遍历 adapter。store 层硬编码 ~/.agents/skills
//     是 OK 的,因为这就是 skillbox 的"全局 Agent"定义(跟 adapter 无关)。

// ResolveGlobalSourcePath 是 resolveGlobalSourcePath 的导出版本,
// 供 controller 层(如 cskill.get_skill / cskill.toggle_global_agent)实时检测
// "skill 是否在 ~/.agents/skills/ 全局目录下"。语义跟 store.buildTreeNode
// 用的判定完全一致,避免出现"列表接口说不是全局,详情接口说是全局"的割裂。
//
// 返回 EvalSymlinks 后的绝对路径,未命中或 stat 失败返空字符串。
func ResolveGlobalSourcePath(name string) string {
	return resolveGlobalSourcePath(name)
}

// resolveGlobalSourcePath 检测 name 对应的 skill 是否在 ~/.agents/skills/ 下。
// 命中时返回 EvalSymlinks 后的绝对路径,未命中或 stat 失败返回空字符串。
func resolveGlobalSourcePath(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	// 候选路径: <home>/.agents/skills/<name>
	candidate := filepath.Join(home, ".agents", "skills", name)
	// EvalSymlinks 解析真实路径(macOS 真实路径在 /private/var/... 下)。
	// 不存在时 EvalSymlinks 也会失败,这时直接 fallback 到 candidate,
	// 反正下面 stat 会判不存在。
	real := candidate
	if r, err := filepath.EvalSymlinks(candidate); err == nil {
		real = r
	}
	// 关键判定:该路径下必须有 SKILL.md 才算"全局 Agent"。
	// 不能只判目录存在 — 避免空目录 / 损坏目录被误识。
	if _, err := os.Stat(filepath.Join(real, "SKILL.md")); err != nil {
		return ""
	}
	return real
}
