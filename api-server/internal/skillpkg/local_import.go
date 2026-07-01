// Package skillpkg - local_import.go
//
// 从本地文件夹 / zip 文件导入 skill 到 skillstore。
//
// 跟现有 Importer.Import 的区别:
//   - Importer.Import 是"扫描已装编程工具的目录 → 选中条目 → store.Save",
//     走的是 skillimporter.Report 流。
//   - 这里用户主动选一个本地文件夹 / zip 包,直接解析 SKILL.md → Canonical
//     → store.Save,不动其它工具。
//
// 校验:目录或 zip 里必须存在 SKILL.md(用户原话要求)。命中数为 0 时返
// ErrNoSkillMD,caller 转 HTTP 400,前端 toast 提示。
package skillpkg

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillimporter"
	"ginp-api/internal/skillstore"
)

// SourceKind 区分本地导入的来源类型,便于前端日志/统计。
type SourceKind string

const (
	SourceFolder    SourceKind = "folder"
	SourceZipPath   SourceKind = "zip_path"
	SourceZipBytes  SourceKind = "zip_bytes"
)

// ErrNoSkillMD 目录或 zip 内未找到任何 SKILL.md。
// 用户需求原文:"导入的时候要检查文件夹内是否存在 SKILL.md 文件"。
var ErrNoSkillMD = errors.New("skillpkg: no SKILL.md found")

// LocalImportResult 一次本地导入的完整产出。
//
// Results 复用 skillimporter.ImportResult(同构),前端可共用渲染。
type LocalImportResult struct {
	Source     string                        `json:"source"`      // 原始路径 / "<zip-bytes>"
	SourceKind SourceKind                    `json:"source_kind"` // folder | zip_path | zip_bytes
	Found      int                           `json:"found"`       // 预检命中的 SKILL.md 数量
	OK         int                           `json:"ok"`          // 成功落地的条数
	Failed     int                           `json:"failed"`      // 失败的条数(含解析失败)
	Results    []skillimporter.ImportResult  `json:"results"`
}

// skillMDName SKILL.md 文件名(常量,避免散落字面量)。
const skillMDName = "SKILL.md"

// ImportFromFolder 递归找 path 下所有"自身含 SKILL.md"的子目录,
// 把每个命中点解析为 Canonical 并走 store.Save。
//
// 行为:
//   - 0 命中 → 返 ErrNoSkillMD。
//   - 单个 SKILL.md 解析失败(无 frontmatter / 无 name)→ 该条 OK=false,不影响整体。
//   - 跳过 symlink 指向目录外的子目录(EvalSymlinks 兜底),避免越界读盘。
func ImportFromFolder(store *skillstore.Store, rootPath string) (*LocalImportResult, error) {
	if store == nil {
		return nil, errors.New("skillpkg: nil store")
	}
	cleaned := filepath.Clean(rootPath)
	fi, err := os.Stat(cleaned)
	if err != nil {
		return nil, fmt.Errorf("skillpkg: stat %s: %w", cleaned, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("skillpkg: not a directory: %s", cleaned)
	}

	out := &LocalImportResult{
		Source:     cleaned,
		SourceKind: SourceFolder,
	}

	// 真实路径:避免 symlink 链造成重复扫描同一根。
	realRoot := cleaned
	if r, err := filepath.EvalSymlinks(cleaned); err == nil {
		realRoot = r
	}

	roots, err := collectSkillDirs(realRoot, maxWalkDepth)
	if err != nil {
		return nil, fmt.Errorf("skillpkg: walk %s: %w", cleaned, err)
	}
	out.Found = len(roots)
	if out.Found == 0 {
		return out, ErrNoSkillMD
	}

	for _, dir := range roots {
		results := importOneFromDir(store, dir)
		out.Results = append(out.Results, results...)
	}
	tallyResults(out)
	return out, nil
}

// ImportFromZipPath 读 zip 文件字节流,转给 ImportFromZipBytes。
func ImportFromZipPath(store *skillstore.Store, zipPath string) (*LocalImportResult, error) {
	if store == nil {
		return nil, errors.New("skillpkg: nil store")
	}
	cleaned := filepath.Clean(zipPath)
	fi, err := os.Stat(cleaned)
	if err != nil {
		return nil, fmt.Errorf("skillpkg: stat %s: %w", cleaned, err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("skillpkg: not a zip file: %s is a directory", cleaned)
	}
	data, err := os.ReadFile(cleaned)
	if err != nil {
		return nil, fmt.Errorf("skillpkg: read %s: %w", cleaned, err)
	}
	out, err := ImportFromZipBytes(store, data)
	if err != nil {
		return out, err
	}
	// 把 Source/SourceKind 覆盖为磁盘路径版,便于前端展示。
	out.Source = cleaned
	out.SourceKind = SourceZipPath
	return out, nil
}

// ImportFromZipBytes 解 zip 字节流,识别所有 SKILL.md 所在目录,逐个落地。
//
// zip 内 SKILL.md 的"skill 根"判定:取 SKILL.md 所在目录(去尾部 /SKILL.md),
// 该目录下所有 entry 作为该 skill 的 Files。
//
// 安全:
//   - 使用 archive/zip 自带的路径解析,跳过目录 entry
//   - 用 path.Clean 校验相对路径不越界(zip slip)
//   - 单个文件大小上限 4 MB(SKILL.md 自身允许任意,文件不超)
//   - 单文件 0 字节 / 解析失败 → 该条 OK=false,其它继续
func ImportFromZipBytes(store *skillstore.Store, zipBytes []byte) (*LocalImportResult, error) {
	if store == nil {
		return nil, errors.New("skillpkg: nil store")
	}
	if len(zipBytes) == 0 {
		return nil, errors.New("skillpkg: empty zip bytes")
	}

	out := &LocalImportResult{
		Source:     "<zip-bytes>",
		SourceKind: SourceZipBytes,
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("skillpkg: open zip: %w", err)
	}

	// 先扫描所有 SKILL.md entry,按"所在目录"分组收集 files。
	bySkillDir, err := groupZipBySkillDir(zr.File)
	if err != nil {
		return nil, err
	}
	out.Found = len(bySkillDir)
	if out.Found == 0 {
		return out, ErrNoSkillMD
	}

	// 排序确保结果顺序稳定(便于测试 + 日志可读)。
	keys := make([]string, 0, len(bySkillDir))
	for k := range bySkillDir {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, dir := range keys {
		entries := bySkillDir[dir]
		results := importOneFromZipEntries(store, dir, entries)
		out.Results = append(out.Results, results...)
	}
	tallyResults(out)
	return out, nil
}

// =================== 内部辅助 ===================

// maxWalkDepth 文件夹递归找 SKILL.md 的最大深度,跟 skillstore/store.go 的
// maxScanDepth 保持同源(8 层)。限制过深是为了防御意外 symlink 链。
const maxWalkDepth = 8

// collectSkillDirs 从 root 出发,WalkDir 收集所有"自身含 SKILL.md"的目录绝对路径。
// 跳过 symlink 指向的目录(避免越界读盘)。
func collectSkillDirs(root string, maxDepth int) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// 单个 entry 出错不中断整体扫描,记录后继续。
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		// 深度限制:用相对路径计算。
		if root != "" && p != root {
			rel, rerr := filepath.Rel(root, p)
			if rerr == nil {
				depth := strings.Count(rel, string(os.PathSeparator)) + 1
				if depth > maxDepth {
					return fs.SkipDir
				}
			}
		}
		// symlink 目录:linux/macOS 上 WalkDir 默认不跟随;此处显式识别并跳过。
		if d.Type()&os.ModeSymlink != 0 {
			return fs.SkipDir
		}
		// 自身有 SKILL.md → 视为 skill 根。
		if _, ferr := os.Stat(filepath.Join(p, skillMDName)); ferr == nil {
			out = append(out, p)
			// 不下钻:Claude marketplaces 偶有"skill 根里再嵌 skill 根"的设计,
			// 这里按"自身有 SKILL.md 即停"处理,语义对齐 skilladapter.BaseAdapter.walkSkills。
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// importOneFromDir 把单个目录里所有文件读到 Canonical,走 store.Save。
func importOneFromDir(store *skillstore.Store, dir string) []skillimporter.ImportResult {
	var results []skillimporter.ImportResult

	canonical, err := readCanonicalFromDir(dir)
	if err != nil {
		results = append(results, skillimporter.ImportResult{
			ToolID: "",
			Name:   filepath.Base(dir),
			OK:     false,
			Error:  err.Error(),
		})
		return results
	}
	if err := store.Save(canonical); err != nil {
		results = append(results, skillimporter.ImportResult{
			ToolID:  "",
			Name:    canonical.Manifest.Name,
			Version: canonical.Manifest.Version,
			OK:      false,
			Error:   err.Error(),
		})
		return results
	}
	results = append(results, skillimporter.ImportResult{
		ToolID:  "",
		Name:    canonical.Manifest.Name,
		Version: canonical.Manifest.Version,
		OK:      true,
	})
	return results
}

// importOneFromZipEntries 把一组 zip entry(同一 skill 根)整合成 Canonical,store.Save。
// entries 中第一个必须是 SKILL.md(其它 file 视为附属)。
func importOneFromZipEntries(store *skillstore.Store, skillDir string, entries []*zip.File) []skillimporter.ImportResult {
	var results []skillimporter.ImportResult
	if len(entries) == 0 {
		return results
	}

	var skillMDEntry *zip.File
	for _, e := range entries {
		if path.Base(e.Name) == skillMDName {
			skillMDEntry = e
			break
		}
	}
	if skillMDEntry == nil {
		// 理论不可能:groupZipBySkillDir 只收集含 SKILL.md 的目录。
		results = append(results, skillimporter.ImportResult{
			ToolID: "",
			Name:   path.Base(skillDir),
			OK:     false,
			Error:  "internal: skill dir without SKILL.md",
		})
		return results
	}

	skillMDContent, err := readZipEntry(skillMDEntry, 4<<20) // SKILL.md 自身允许到 4 MB
	if err != nil {
		results = append(results, skillimporter.ImportResult{
			ToolID: "",
			Name:   path.Base(skillDir),
			OK:     false,
			Error:  fmt.Sprintf("read SKILL.md: %v", err),
		})
		return results
	}

	canonical, err := skilladapter.ParseSkillMD(string(skillMDContent))
	if err != nil {
		results = append(results, skillimporter.ImportResult{
			ToolID: "",
			Name:   path.Base(skillDir),
			OK:     false,
			Error:  fmt.Sprintf("parse SKILL.md: %v", err),
		})
		return results
	}

	// 加载其它附属 files。
	for _, e := range entries {
		if e == skillMDEntry {
			continue
		}
		// entry 相对 skill 根的相对路径(zip 内用 "/",因此 base = path.Base)。
		rel := strings.TrimPrefix(e.Name, skillDir)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" || rel == skillMDName {
			continue
		}
		data, err := readZipEntry(e, 4<<20)
		if err != nil {
			// 单文件失败不让整条 skill 报废,记录到 Manifest 之外不方便,
			// 这里直接 abort 当前 skill,让 caller 看到 Error。
			results = append(results, skillimporter.ImportResult{
				ToolID:  "",
				Name:    canonical.Manifest.Name,
				Version: canonical.Manifest.Version,
				OK:      false,
				Error:   fmt.Sprintf("read %s: %v", e.Name, err),
			})
			return results
		}
		canonical.Files = append(canonical.Files, skilladapter.File{
			Path:    filepath.ToSlash(rel),
			Content: string(data),
		})
	}

	if err := store.Save(*canonical); err != nil {
		results = append(results, skillimporter.ImportResult{
			ToolID:  "",
			Name:    canonical.Manifest.Name,
			Version: canonical.Manifest.Version,
			OK:      false,
			Error:   err.Error(),
		})
		return results
	}
	results = append(results, skillimporter.ImportResult{
		ToolID:  "",
		Name:    canonical.Manifest.Name,
		Version: canonical.Manifest.Version,
		OK:      true,
	})
	return results
}

// groupZipBySkillDir 把 zip 内 entry 按"SKILL.md 所在目录"分组。
// 返回:map[skillDir][]*zip.File,skillDir 是 zip 内的相对路径(如 "skills/foo")。
func groupZipBySkillDir(files []*zip.File) (map[string][]*zip.File, error) {
	out := map[string][]*zip.File{}
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		// 安全校验:路径不越界(防 zip slip)。zip 内部 path 已用 "/"。
		cleaned := path.Clean(f.Name)
		if cleaned == "." || strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/../") {
			continue
		}
		dir := path.Dir(cleaned)
		base := path.Base(cleaned)
		if base != skillMDName {
			// 附属文件:归到它所在目录对应的 skill 根(若该目录或其祖先有 SKILL.md)。
			// 这里采用"扁平归组":任何 file 归到 top-level skill 根。
			// 简化:把 file 放进它父目录,后续 importOneFromZipEntries 再按
			// skill 根过滤(同 skill 根的 file 才会被采纳)。
			out[dir] = append(out[dir], f)
			continue
		}
		// SKILL.md:把它和 dir 下所有 file 一起归到 dir 这个 skill 根。
		out[dir] = append(out[dir], f)
	}
	// 把"没有 SKILL.md"的目录 entry 过滤掉 —— 即只保留含 SKILL.md 的目录。
	filtered := map[string][]*zip.File{}
	for dir, es := range out {
		hasMD := false
		for _, e := range es {
			if path.Base(e.Name) == skillMDName {
				hasMD = true
				break
			}
		}
		if hasMD {
			filtered[dir] = es
		}
	}
	return filtered, nil
}

// readZipEntry 安全读 zip 单文件内容,带大小上限。
func readZipEntry(f *zip.File, max int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", f.Name, err)
	}
	defer rc.Close()

	// 优先用限流 reader,避免恶意 zip entry 占用大量内存。
	lr := io.LimitReader(rc, max+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", f.Name, err)
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("%s too large: %d bytes (limit %d)", f.Name, len(data), max)
	}
	return data, nil
}

// readCanonicalFromDir 把目录里所有文件读到 Canonical。
// 对齐 skilladapter.readSkillDir 的语义:必须含 SKILL.md,frontmatter 校验由
// ParseSkillMD 完成,失败整条失败。
func readCanonicalFromDir(dir string) (skilladapter.Canonical, error) {
	skillMDPath := filepath.Join(dir, skillMDName)
	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return skilladapter.Canonical{}, fmt.Errorf("read SKILL.md: %w", err)
	}
	canonical, err := skilladapter.ParseSkillMD(string(content))
	if err != nil {
		return skilladapter.Canonical{}, fmt.Errorf("parse SKILL.md: %w", err)
	}
	// 加载附属 files:跳过 SKILL.md(已含在 manifest),用 EvalSymlinks 解决 symlink 链。
	realDir := dir
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		realDir = r
	}
	err = filepath.WalkDir(realDir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if path.Base(p) == skillMDName {
			return nil
		}
		rel, rerr := filepath.Rel(realDir, p)
		if rerr != nil {
			return nil
		}
		// 跳过 symlink 文件
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		canonical.Files = append(canonical.Files, skilladapter.File{
			Path:    filepath.ToSlash(rel),
			Content: string(data),
		})
		return nil
	})
	if err != nil {
		return skilladapter.Canonical{}, fmt.Errorf("walk %s: %w", dir, err)
	}
	return *canonical, nil
}

// tallyResults 统计 ok / failed。
func tallyResults(out *LocalImportResult) {
	for _, r := range out.Results {
		if r.OK {
			out.OK++
		} else {
			out.Failed++
		}
	}
}