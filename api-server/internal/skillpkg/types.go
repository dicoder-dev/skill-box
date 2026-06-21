// Package skillpkg 提供 skill 的"导出 / 导入"能力,把一个或多个 canonical skill
// 打包成一个 .skillbox 文件(zip 容器)便于跨机器分享 / 备份。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.5 节):
//   - .skillbox 就是一个 zip,根目录有 manifest.json(包级元数据)+ skills/<name>@<version>/ 子目录
//   - 每个子目录里又是 zip-of-zip 的平铺:SKILL.md / skill.yaml(manifest) / 其他文件
//   - 导入时:解析包级 manifest → 选要装的 skill → 走 sskill.Service.Create 写盘 + DB
//   - 导出时:走 sskill.Service.GetFull(单条) / 批量读取后打 zip
package skillpkg

import (
	"errors"
)

// 文件扩展名。
const Ext = ".skillbox"

// 包内 manifest 文件名(放 zip 根)。
const manifestFile = "manifest.json"

// 业务错误。
var (
	ErrEmptySkills      = errors.New("skillpkg: no skills to package")
	ErrInvalidManifest  = errors.New("skillpkg: invalid package manifest")
	ErrUnknownSkillKey  = errors.New("skillpkg: skill key not found in package")
	ErrInvalidSkillMeta = errors.New("skillpkg: invalid skill meta in package")
)

// PackageManifest .skillbox 包的元数据(根 manifest.json 内容)。
type PackageManifest struct {
	PkgFormat  string         `json:"pkg_format"`  // 当前固定 "skillbox.v1"
	CreatedAt  string         `json:"created_at"`  // RFC3339
	SourceApp  string         `json:"source_app"`  // 来自哪个 Skill Box 实例(可空)
	SourceDesc string         `json:"source_desc"` // 备注
	Skills     []SkillSummary `json:"skills"`      // 包内全部 skill 的索引
}

// SkillSummary 索引里每条 skill 的轻量元数据。
type SkillSummary struct {
	Key       string `json:"key"`        // "name@version",与 import 时的引用一致
	Name      string `json:"name"`
	Version   string `json:"version"`
	Scope     string `json:"scope"`      // 默认 target scope(可被 import 入参覆盖)
	ProjectID uint   `json:"project_id"` // scope=project 时用
	Author    string `json:"author,omitempty"`
	License   string `json:"license,omitempty"`
}

// ExportRequest 导出端入参。
type ExportRequest struct {
	Skills    []SkillRef `json:"skills"`              // 至少 1 个
	SourceApp string     `json:"source_app,omitempty"` // 来源(可选)
	SourceDesc string    `json:"source_desc,omitempty"`
}

// SkillRef 标识一个要导出的 skill。
type SkillRef struct {
	Scope     string `json:"scope"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

// ImportRequest 导入端入参。
// Skills 为空 = 装包内全部;否则按 Key 装指定子集。
type ImportRequest struct {
	SourceDesc string             `json:"source_desc,omitempty"` // 可选备注
	TargetScope string            `json:"target_scope"`          // global | project
	ProjectID  uint               `json:"project_id"`            // target_scope=project 时必填
	Skills     []ImportSkillEntry `json:"skills,omitempty"`      // 空 = 全部
}

// ImportSkillEntry 单个要装的 skill。
type ImportSkillEntry struct {
	Key        string `json:"key"`                   // "name@version"
	TargetScope string `json:"target_scope,omitempty"` // 覆盖包级 target_scope
	ProjectID  uint   `json:"project_id,omitempty"`   // 同上
}

// ImportResult 导入结果汇总。
type ImportResult struct {
	ImportedAt string                 `json:"imported_at"`
	Total      int                    `json:"total"`
	OK         int                    `json:"ok"`
	Failed     int                    `json:"failed"`
	Items      []ImportResultItem     `json:"items"`
}

// ImportResultItem 单条导入结果。
type ImportResultItem struct {
	Key      string `json:"key"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	SkillID  uint   `json:"skill_id,omitempty"`
}
