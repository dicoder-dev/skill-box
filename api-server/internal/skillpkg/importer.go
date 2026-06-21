// skillpkg/importer.go - 解析 .skillbox zip,选要装的 skill,走 provider 装入。
package skillpkg

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"ginp-api/internal/skilladapter"
)

// Importer 解析 + 装入的总入口;不依赖 controller 层(由 sskillpkg 包去 wire)。
type Importer struct {
	// Provider 把单个 canonical 装入 store(scope+project 已定)。
	// 真实实现走 sskill.Service.Create(已有回滚能力)。
	Provider SkillInstaller
	// Now 注入便于测试;nil 时用 time.Now().UTC()。
	Now func() time.Time
}

// SkillInstaller 暴露给 importer 的最小接口。
type SkillInstaller interface {
	InstallCanonical(scope string, projectID uint, c skilladapter.Canonical, source string) (uint, error)
}

// NewImporter 构造器。
func NewImporter(p SkillInstaller) *Importer {
	return &Importer{Provider: p, Now: func() time.Time { return time.Now().UTC() }}
}

// ParseManifest 解析 zip 字节流,只读 manifest.json(用于前端预览包内容)。
func ParseManifest(zipBytes []byte) (*PackageManifest, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("skillpkg: open zip: %w", err)
	}
	for _, f := range zr.File {
		if path.Clean(f.Name) != manifestFile {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("skillpkg: open manifest: %w", err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("skillpkg: read manifest: %w", err)
		}
		var m PackageManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		if m.PkgFormat == "" {
			return nil, fmt.Errorf("%w: missing pkg_format", ErrInvalidManifest)
		}
		return &m, nil
	}
	return nil, fmt.Errorf("%w: manifest.json not found", ErrInvalidManifest)
}

// Install 完整流程:解析 → 选条目 → 装入 → 返回结果。
func (i *Importer) Install(zipBytes []byte, req ImportRequest) (*ImportResult, error) {
	if req.TargetScope != skilladapter.ScopeGlobal && req.TargetScope != skilladapter.ScopeProject {
		return nil, fmt.Errorf("skillpkg: invalid target_scope %q", req.TargetScope)
	}
	if req.TargetScope == skilladapter.ScopeProject && req.ProjectID == 0 {
		return nil, fmt.Errorf("skillpkg: project_id required when target_scope=project")
	}

	pkg, skills, err := i.parseAll(zipBytes)
	if err != nil {
		return nil, err
	}

	// 决定要装哪些 key
	keys := make([]string, 0, len(skills))
	if len(req.Skills) == 0 {
		// 全部
		for k := range skills {
			keys = append(keys, k)
		}
	} else {
		for _, e := range req.Skills {
			keys = append(keys, e.Key)
		}
	}

	now := i.now().Format(time.RFC3339)
	out := &ImportResult{ImportedAt: now}
	for _, k := range keys {
		c, ok := skills[k]
		if !ok {
			out.Items = append(out.Items, ImportResultItem{Key: k, OK: false, Error: ErrUnknownSkillKey.Error()})
			out.Failed++
			continue
		}
		scope, projID := req.TargetScope, req.ProjectID
		// 单条覆盖
		for _, e := range req.Skills {
			if e.Key == k {
				if e.TargetScope == skilladapter.ScopeGlobal || e.TargetScope == skilladapter.ScopeProject {
					scope = e.TargetScope
				}
				if e.ProjectID != 0 {
					projID = e.ProjectID
				}
				break
			}
		}

		id, err := i.Provider.InstallCanonical(scope, projID, c, "pkg:"+pkg.PkgFormat)
		if err != nil {
			out.Items = append(out.Items, ImportResultItem{Key: k, OK: false, Error: err.Error()})
			out.Failed++
			continue
		}
		out.Items = append(out.Items, ImportResultItem{Key: k, OK: true, SkillID: id})
		out.OK++
	}
	out.Total = len(out.Items)
	return out, nil
}

// parseAll 解析整个包,返回 manifest + map[key]Canonical。
// 这里把整个包解析为内存结构;P1 如果遇到大文件可改成"延迟读 + 流式装"。
func (i *Importer) parseAll(zipBytes []byte) (*PackageManifest, map[string]skilladapter.Canonical, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, nil, fmt.Errorf("skillpkg: open zip: %w", err)
	}

	// 先收所有 entry 到 map:dirPrefix → map[相对路径]bytes
	byDir := map[string]map[string][]byte{}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// 期望结构: skills/<key>/...  或  manifest.json
		name := path.Clean(f.Name)
		if name == manifestFile {
			continue
		}
		if !strings.HasPrefix(name, "skills/") {
			continue
		}
		rest := strings.TrimPrefix(name, "skills/")
		slash := strings.Index(rest, "/")
		if slash < 0 {
			continue // 顶层文件,跳过
		}
		key := rest[:slash]
		rel := rest[slash+1:]
		if _, ok := byDir[key]; !ok {
			byDir[key] = map[string][]byte{}
		}
		rc, err := f.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("skillpkg: open %s: %w", name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("skillpkg: read %s: %w", name, err)
		}
		byDir[key][rel] = data
	}

	// 解析 manifest
	pkg, err := i.parseManifest(zr)
	if err != nil {
		return nil, nil, err
	}

	out := make(map[string]skilladapter.Canonical, len(byDir))
	for key, files := range byDir {
		manifestData, ok := files["skill.yaml"]
		if !ok {
			return nil, nil, fmt.Errorf("%w: %s missing skill.yaml", ErrInvalidSkillMeta, key)
		}
		var m skilladapter.Manifest
		if err := yaml.Unmarshal(manifestData, &m); err != nil {
			return nil, nil, fmt.Errorf("%w: %s parse: %v", ErrInvalidSkillMeta, key, err)
		}
		if m.Name == "" || m.Version == "" {
			return nil, nil, fmt.Errorf("%w: %s missing name/version", ErrInvalidSkillMeta, key)
		}
		canonical := skilladapter.Canonical{Manifest: m}
		for rel, data := range files {
			if rel == "skill.yaml" {
				continue
			}
			canonical.Files = append(canonical.Files, skilladapter.File{Path: rel, Content: string(data)})
		}
		out[key] = canonical
	}

	// 校验:每个 manifest.Skills.Key 都应该解析出条目
	for _, s := range pkg.Skills {
		if _, ok := out[s.Key]; !ok {
			return nil, nil, fmt.Errorf("%w: manifest 声明 %s 但包内缺失", ErrInvalidManifest, s.Key)
		}
	}

	return pkg, out, nil
}

func (i *Importer) parseManifest(zr *zip.Reader) (*PackageManifest, error) {
	for _, f := range zr.File {
		if path.Clean(f.Name) != manifestFile {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("skillpkg: open manifest: %w", err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("skillpkg: read manifest: %w", err)
		}
		var m PackageManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		if m.PkgFormat == "" {
			return nil, fmt.Errorf("%w: missing pkg_format", ErrInvalidManifest)
		}
		return &m, nil
	}
	return nil, fmt.Errorf("%w: manifest.json not found", ErrInvalidManifest)
}

func (i *Importer) now() time.Time {
	if i.Now != nil {
		return i.Now()
	}
	return time.Now().UTC()
}

// errIs 是 errors.Is 的本地别名(避免每次都 import errors)
func errIs(err, target error) bool {
	return errors.Is(err, target)
}
