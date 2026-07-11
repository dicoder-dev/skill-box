// Package skillpkg - local_import.go
//
// 从本地文件夹 / 压缩包导入 skill 到 skillstore。
//
// 跟现有 Importer.Import 的区别:
//   - Importer.Import 是"扫描已装编程工具的目录 → 选中条目 → store.Save",
//     走的是 skillimporter.Report 流。
//   - 这里用户主动选一个本地文件夹 / 压缩包,直接解析 SKILL.md → Canonical
//     → store.Save,不动其它工具。
//
// 校验:目录或压缩包内必须存在 SKILL.md(用户原话要求)。命中数为 0 时返
// ErrNoSkillMD,caller 转 HTTP 400,前端 toast 提示。
//
// 2026-07-11 增:支持多种压缩格式(.zip / .tar / .tar.gz / .tgz / .tar.bz2 /
// .tbz2 / .tar.xz / .txz),底层走 Go 标准库 archive/tar + compress/*。
// 7z / rar 不在范围内(需第三方库,见任务讨论)。
package skillpkg

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
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

	xz "github.com/ulikunitz/xz"
)

// SourceKind 区分本地导入的来源类型,便于前端日志/统计。
type SourceKind string

const (
	SourceFolder   SourceKind = "folder"
	SourceArchive  SourceKind = "archive" // zip / tar / tar.* (旧 zip_path / zip_bytes 合并)
	SourceArchiveB SourceKind = "archive_bytes"
	// SourceGlobalPaths 2026-07-10 增:全局目录批量导入(走 ImportFromPaths)。
	SourceGlobalPaths SourceKind = "global_paths"
)

// 兼容旧字段:沿用旧名字,避免前端 JSON 解析崩。
const (
	SourceZipPath  = SourceArchive
	SourceZipBytes = SourceArchiveB
)

// ErrNoSkillMD 目录或压缩包内未找到任何 SKILL.md。
// 用户需求原文:"导入的时候要检查文件夹内是否存在 SKILL.md 文件"。
var ErrNoSkillMD = errors.New("skillpkg: no SKILL.md found")

// ErrUnsupportedArchive 压缩包格式不在支持范围内(非 zip/tar.*)。
// 前端可针对此错弹"暂不支持 .xxx 格式,支持 zip/tar/tar.gz/tar.bz2/tar.xz"。
var ErrUnsupportedArchive = errors.New("skillpkg: unsupported archive format (supported: zip, tar, tar.gz, tgz, tar.bz2, tbz2, tar.xz, txz)")

// LocalImportResult 一次本地导入的完整产出。
//
// Results 复用 skillimporter.ImportResult(同构),前端可共用渲染。
type LocalImportResult struct {
	Source     string                       `json:"source"`      // 原始路径 / "<archive-bytes>"
	SourceKind SourceKind                   `json:"source_kind"` // folder | archive | archive_bytes
	Found      int                          `json:"found"`       // 预检命中的 SKILL.md 数量
	OK         int                          `json:"ok"`          // 成功落地的条数
	Failed     int                          `json:"failed"`      // 失败的条数(含解析失败)
	Results    []skillimporter.ImportResult `json:"results"`
}

// skillMDName SKILL.md 文件名(常量,避免散落字面量)。
const skillMDName = "SKILL.md"

// archiveEntry 是从压缩包读取出来的"统一条目"。
//
// 设计目的:把 zip.File / tar.Header 两种 entry 抽象成同一种结构,后续 SKILL.md
// 检索逻辑只关心 name + reader,不关心原始格式。
//
// 2026-07-11 改:name 字段在 entry 入库时统一规范为 POSIX 风格(正斜杠 + 大写 SKILL.md
// 匹配)。Windows 风格 zip 用反斜杠做分隔符,如果不归一会导致 path.Base/Dir 把整条
// 路径当成 basename,SKILL.md 检测失败(用户报 deepwhite-*.zip 导入失败就是这个原因)。
type archiveEntry struct {
	// name:压缩包内的相对路径,统一用正斜杠(如 "skills/foo/SKILL.md")
	name string
	// size:未压缩字节数,-1 表示未知(tar 里有些 entry 没法直接读到 size)
	size int64
	// open:返回该 entry 的字节流 reader,由调用方负责 close。
	open func() (io.ReadCloser, error)
}

// normalizeArchivePath 把 Windows 风格分隔符(反斜杠)统一成 POSIX(正斜杠),
// 顺便清理连续斜杠。tar / zip 在 Windows / macOS / Linux 行为不一致:
//   - macOS / Linux zip 默认正斜杠
//   - Windows zip 可能反斜杠(尤其右键压缩的)
//   - tar 始终正斜杠
//
// 在 entry 入库前一次性归一化,后续 path.Base/Dir/Clean 不需要再操心。
func normalizeArchivePath(name string) string {
	name = strings.ReplaceAll(name, `\`, `/`)
	// 清理 // → /,避免 "./" 等冗余形态干扰 Dir 计算。
	for strings.Contains(name, "//") {
		name = strings.ReplaceAll(name, "//", "/")
	}
	return name
}

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

// ImportFromArchivePath 读磁盘上的压缩包(支持 zip / tar / tar.gz / tar.bz2 / tar.xz),
// 转给 ImportFromArchiveBytes。文件后缀 + magic bytes 双重判定,防止有人把 zip 改名为 .tar。
func ImportFromArchivePath(store *skillstore.Store, archivePath string) (*LocalImportResult, error) {
	if store == nil {
		return nil, errors.New("skillpkg: nil store")
	}
	cleaned := filepath.Clean(archivePath)
	fi, err := os.Stat(cleaned)
	if err != nil {
		return nil, fmt.Errorf("skillpkg: stat %s: %w", cleaned, err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("skillpkg: not an archive file: %s is a directory", cleaned)
	}
	data, err := os.ReadFile(cleaned)
	if err != nil {
		return nil, fmt.Errorf("skillpkg: read %s: %w", cleaned, err)
	}
	out, err := ImportFromArchiveBytes(store, data)
	if err != nil {
		return out, err
	}
	// 把 Source/SourceKind 覆盖为磁盘路径版,便于前端展示。
	out.Source = cleaned
	out.SourceKind = SourceArchive
	return out, nil
}

// ImportFromZipPath 向后兼容别名,内部转调 ImportFromArchivePath。
//
// 2026-07-11:旧前端 / 旧 controller 还在用 ImportFromZipPath,保留兼容。
func ImportFromZipPath(store *skillstore.Store, zipPath string) (*LocalImportResult, error) {
	return ImportFromArchivePath(store, zipPath)
}

// ImportFromArchiveBytes 解压缩包字节流,识别所有 SKILL.md 所在目录,逐个落地。
//
// 支持的格式:
//   - zip(archive/zip)
//   - tar(archive/tar)
//   - tar.gz / tgz(compress/gzip)
//   - tar.bz2 / tbz2( compress/bzip2,需手动包 Reader)
//   - tar.xz / txz(github.com/ulikunitz/xz,纯 Go,无 C 依赖)
//
// 判定策略:扩展名优先,magic bytes 兜底。两者冲突时以 magic bytes 为准
// (防止有人把 .zip 改成 .tar 骗服务)。
//
// zip 内 SKILL.md 的"skill 根"判定:取 SKILL.md 所在目录(去尾部 /SKILL.md),
// 该目录下所有 entry 作为该 skill 的 Files。
//
// 安全:
//   - 使用 archive/zip + archive/tar 自带的路径解析,跳过目录 entry
//   - 用 path.Clean 校验相对路径不越界(zip / tar slip)
//   - 单个文件大小上限 4 MB(SKILL.md 自身允许任意,文件不超)
//   - 单文件 0 字节 / 解析失败 → 该条 OK=false,其它继续
func ImportFromArchiveBytes(store *skillstore.Store, archiveBytes []byte) (*LocalImportResult, error) {
	if store == nil {
		return nil, errors.New("skillpkg: nil store")
	}
	if len(archiveBytes) == 0 {
		return nil, errors.New("skillpkg: empty archive bytes")
	}

	out := &LocalImportResult{
		Source:     "<archive-bytes>",
		SourceKind: SourceArchiveB,
	}

	entries, err := readArchiveEntries(archiveBytes)
	if err != nil {
		return nil, err
	}

	// 先扫描所有 SKILL.md entry,按"所在目录"分组收集 files。
	bySkillDir, err := groupEntriesBySkillDir(entries)
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
		es := bySkillDir[dir]
		results := importOneFromArchiveEntries(store, dir, es)
		out.Results = append(out.Results, results...)
	}
	tallyResults(out)
	return out, nil
}

// ImportFromZipBytes 向后兼容别名,内部转调 ImportFromArchiveBytes。
//
// 2026-07-11:旧前端 / 旧 controller 还在用 ImportFromZipBytes,保留兼容。
func ImportFromZipBytes(store *skillstore.Store, zipBytes []byte) (*LocalImportResult, error) {
	return ImportFromArchiveBytes(store, zipBytes)
}

// =================== 内部辅助 ===================

// maxWalkDepth 文件夹递归找 SKILL.md 的最大深度,跟 skillstore/store.go 的
// maxScanDepth 保持同源(8 层)。限制过深是为了防御意外 symlink 链。
const maxWalkDepth = 8

// 单文件 4MB 上限。SKILL.md 一般几 KB,4MB 留足余量。
const maxEntryBytes = 4 << 20

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

// importOneFromArchiveEntries 把一组 archive entry(同一 skill 根)整合成 Canonical,store.Save。
// entries 中第一个必须是 SKILL.md(其它 file 视为附属)。
func importOneFromArchiveEntries(store *skillstore.Store, skillDir string, entries []archiveEntry) []skillimporter.ImportResult {
	var results []skillimporter.ImportResult
	if len(entries) == 0 {
		return results
	}

	var skillMDEntry *archiveEntry
	for i := range entries {
		if path.Base(entries[i].name) == skillMDName {
			skillMDEntry = &entries[i]
			break
		}
	}
	if skillMDEntry == nil {
		// 理论不可能:groupEntriesBySkillDir 只收集含 SKILL.md 的目录。
		results = append(results, skillimporter.ImportResult{
			ToolID: "",
			Name:   path.Base(skillDir),
			OK:     false,
			Error:  "internal: skill dir without SKILL.md",
		})
		return results
	}

	skillMDContent, err := readArchiveEntry(*skillMDEntry, maxEntryBytes)
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
		if e.name == skillMDEntry.name {
			continue
		}
		// entry 相对 skill 根的相对路径(archive 内用 "/",因此 base = path.Base)。
		rel := strings.TrimPrefix(e.name, skillDir)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" || rel == skillMDName {
			continue
		}
		data, err := readArchiveEntry(e, maxEntryBytes)
		if err != nil {
			// 单文件失败不让整条 skill 报废,记录到 Manifest 之外不方便,
			// 这里直接 abort 当前 skill,让 caller 看到 Error。
			results = append(results, skillimporter.ImportResult{
				ToolID:  "",
				Name:    canonical.Manifest.Name,
				Version: canonical.Manifest.Version,
				OK:      false,
				Error:   fmt.Sprintf("read %s: %v", e.name, err),
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

// groupEntriesBySkillDir 把 archive entry 按"SKILL.md 所在目录"分组。
// 返回:map[skillDir][]archiveEntry,skillDir 是 archive 内的相对路径(如 "skills/foo")。
//
// 2026-07-11 重写分组逻辑:
//
// 旧实现按 entry 自身 Dir 分组,有问题 —— zip 里 SKILL.md 在子目录顶层,
// 附属 file 在更深的子目录,两者 dir 不同,被拆成两个 group,导致附属 file 被丢。
// 真实案例:deepwhite-screenwriting-v1\SKILL.md vs deepwhite-screenwriting-v1\references\engine.md。
//
// 新实现:对每个 entry,沿"自身目录 → 父目录 → ... → 根"找最近的含 SKILL.md 的祖先,
// 归到那个祖先目录。这样 SKILL.md 和它的附属 file(无论嵌套多深)都正确归一组。
func groupEntriesBySkillDir(entries []archiveEntry) (map[string][]archiveEntry, error) {
	// 先建索引:含 SKILL.md 的所有目录路径(用于祖先查找)。
	skillDirs := map[string]bool{}
	for _, e := range entries {
		// 安全校验:路径不越界(防 archive slip)。
		cleaned := path.Clean(e.name)
		if cleaned == "." || strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/../") {
			continue
		}
		if path.Base(cleaned) == skillMDName {
			skillDirs[path.Dir(cleaned)] = true
		}
	}

	out := map[string][]archiveEntry{}
	for _, e := range entries {
		cleaned := path.Clean(e.name)
		if cleaned == "." || strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/../") {
			continue
		}
		// 沿目录链向上找最近的 skillDir。
		d := path.Dir(cleaned)
		var ancestor string
		for d != "." && d != "/" && d != "" {
			if skillDirs[d] {
				ancestor = d
				break
			}
			d = path.Dir(d)
		}
		// 顶层 SKILL.md(目录 = ".")的情况
		if ancestor == "" && skillDirs["."] {
			ancestor = "."
		}
		if ancestor == "" {
			continue
		}
		out[ancestor] = append(out[ancestor], e)
	}
	return out, nil
}

// readArchiveEntry 安全读 archive 单文件内容,带大小上限。
func readArchiveEntry(e archiveEntry, max int64) ([]byte, error) {
	rc, err := e.open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", e.name, err)
	}
	defer rc.Close()

	// 优先用限流 reader,避免恶意 entry 占用大量内存。
	lr := io.LimitReader(rc, max+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", e.name, err)
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("%s too large: %d bytes (limit %d)", e.name, len(data), max)
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

// =================== 压缩包格式解析 ===================

// archiveKind 压缩包格式。
type archiveKind int

const (
	archiveUnknown archiveKind = iota
	archiveZip
	archiveTar          // 纯 tar(不压缩)
	archiveTarGz        // tar + gzip
	archiveTarBz2       // tar + bzip2
	archiveTarXz        // tar + xz
)

// 压缩包 magic bytes,用于格式识别(防扩展名被骗)。
//
//   - zip:   PK\x03\x04(local file header)
//   - gzip:  \x1f\x8b
//   - bzip2: BZ(h)
//   - xz:    \xfd7zXZ\x00(标准 xz magic)
//   - tar:   没固定 magic(ustar 在 offset 257),只能靠扩展名 + 非上述 magic 兜底
var (
	gzipMagic  = []byte{0x1f, 0x8b}
	bzip2Magic = []byte{'B', 'Z', 'h'}
	xzMagic    = []byte{0xfd, '7', 'z', 'X', 'Z', 0x00}
	zipMagic   = []byte{'P', 'K', 0x03, 0x04}
)

// detectArchiveKind 优先看 magic bytes,扩展名兜底。
//
// 防御场景:用户把 .zip 改成 .tar.gz,期望走 zip 解压。
// 行为:magic bytes 一旦匹配就以 magic 为准,扩展名只用于 magic 不够唯一的场景(tar / tar.*)。
//
// tar 特殊说明:纯 tar 没固定文件头 magic,只在每个 512 字节 header 的 offset 257 处
// 有 "ustar\0" 标识(USTAR/POSIX/GNU 都遵循)。如果 magic 都没命中,再走扩展名兜底。
func detectArchiveKind(data []byte, filename string) archiveKind {
	if len(data) >= 4 && bytes.HasPrefix(data, zipMagic) {
		return archiveZip
	}
	if len(data) >= 2 && bytes.HasPrefix(data, gzipMagic) {
		return archiveTarGz
	}
	if len(data) >= 3 && bytes.HasPrefix(data, bzip2Magic) {
		return archiveTarBz2
	}
	if len(data) >= 6 && bytes.HasPrefix(data, xzMagic) {
		return archiveTarXz
	}
	// magic 不命中 → 看扩展名兜底(纯 tar / 各种变体)。
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return archiveTarGz
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz2"):
		return archiveTarBz2
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"):
		return archiveTarXz
	case strings.HasSuffix(lower, ".tar"):
		return archiveTar
	}
	// 还没命中 → 探一下 tar 的 ustar magic(offset 257)。
	// 这是 tar 文件 header 末尾的标准 magic,即便没扩展名也能识别。
	if hasUSTARMagic(data) {
		return archiveTar
	}
	return archiveUnknown
}

// hasUSTARMagic 探 tar USTAR header(每个 512 字节 header 的 offset 257 处有 "ustar\0")。
func hasUSTARMagic(data []byte) bool {
	if len(data) < 262 {
		return false
	}
	return bytes.Equal(data[257:263], []byte("ustar\x00"))
}

// readArchiveEntries 把压缩包字节流展开成统一 []archiveEntry。
// 失败时返 ErrUnsupportedArchive(其它格式未识别)。
func readArchiveEntries(data []byte) ([]archiveEntry, error) {
	kind := detectArchiveKind(data, "")
	switch kind {
	case archiveZip:
		return readZipEntries(data)
	case archiveTar:
		return readTarEntries(bytes.NewReader(data))
	case archiveTarGz:
		return readTarGzEntries(data)
	case archiveTarBz2:
		return readTarBz2Entries(data)
	case archiveTarXz:
		return readTarXZEntries(data)
	default:
		return nil, ErrUnsupportedArchive
	}
}

// readZipEntries 把 zip 字节流展开成 []archiveEntry。
func readZipEntries(data []byte) ([]archiveEntry, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("skillpkg: open zip: %w", err)
	}
	out := make([]archiveEntry, 0, len(zr.File))
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// 闭包捕获 f,每次 open 都返回新的 reader。
		f := f
		out = append(out, archiveEntry{
			name: normalizeArchivePath(f.Name),
			size: int64(f.UncompressedSize64),
			open: func() (io.ReadCloser, error) { return f.Open() },
		})
	}
	return out, nil
}

// readTarEntries 把纯 tar 字节流展开成 []archiveEntry。
//
// tar 流式 reader 特性:Next() 之后 payload 只能从 tr 读一次;后面再访问 tr 已
// 经被消耗。所以这里把 payload 立刻缓冲进 []byte(受 maxEntryBytes 限流),
// open 时直接 bytes.NewReader,语义跟 zip.File.Open() 一致,支持"延迟 open"。
//
// 限制:tar 整体内容会一次性解压到内存(gzip/bzip2/xz 都在外层完成),
// 大文件解压对内存压力较大。当前 use case 是导入 skill,单包几 MB 量级,够用。
func readTarEntries(r io.Reader) ([]archiveEntry, error) {
	tr := tar.NewReader(r)
	var out []archiveEntry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("skillpkg: read tar header: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		// 限流读 payload:超 maxEntryBytes 直接报错,避免恶意包耗内存。
		// 注意 limit = max + 1,读出来再判断是否超限。
		lr := io.LimitReader(tr, maxEntryBytes+1)
		payload, err := io.ReadAll(lr)
		if err != nil {
			return nil, fmt.Errorf("skillpkg: read tar entry %s: %w", hdr.Name, err)
		}
		if int64(len(payload)) > maxEntryBytes {
			return nil, fmt.Errorf("tar entry %s too large: %d bytes (limit %d)", hdr.Name, len(payload), maxEntryBytes)
		}
		// payload 复制一份闭包捕获,避免覆盖。
		buf := payload
		hdrName := hdr.Name
		out = append(out, archiveEntry{
			name: normalizeArchivePath(hdrName),
			size: int64(len(buf)),
			open: func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(buf)), nil
			},
		})
	}
	return out, nil
}

// readTarGzEntries tar.gz = gzip + tar。
func readTarGzEntries(data []byte) ([]archiveEntry, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("skillpkg: open gzip: %w", err)
	}
	defer gz.Close()
	return readTarEntries(gz)
}

// readTarBz2Entries tar.bz2 = bzip2 + tar。
//
// 注意:compress/bzip2 没有现成 NewReader 包 Reader,自己拼一层(标准库就这德性)。
func readTarBz2Entries(data []byte) ([]archiveEntry, error) {
	bz := bzip2.NewReader(bytes.NewReader(data))
	return readTarEntries(bz)
}

// readTarXZEntries tar.xz = xz + tar。
//
// 用 github.com/ulikunitz/xz,纯 Go,无 C 依赖。已经在很多生产环境用,稳定。
func readTarXZEntries(data []byte) ([]archiveEntry, error) {
	xr, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("skillpkg: open xz: %w", err)
	}
	return readTarEntries(xr)
}