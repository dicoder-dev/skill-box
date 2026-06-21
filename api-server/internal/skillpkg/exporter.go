// skillpkg/exporter.go - 把一个或多个 canonical skill 打包成 .skillbox。
package skillpkg

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"ginp-api/internal/skilladapter"
)

// CanonicalProvider 暴露给 exporter 的最小接口,避免直接依赖 sskill(防循环)。
// 实现方:service 层把"按 SkillRef 拿 Canonical"包装成此接口。
type CanonicalProvider interface {
	// LoadCanonical 返回 (canonical, ok);不存在或读失败时 ok=false。
	// caller 自己根据 ok 决定要不要计入失败列表。
	LoadCanonical(scope string, projectID uint, name, version string) (skilladapter.Canonical, bool, error)
}

// BuildBytes 在内存里生成 .skillbox 的 zip 字节流,便于直接返回给 HTTP。
//
// skills: 至少 1 个;每个 ref 在 pack 里都要能找到对应 canonical,否则整包失败。
// 返回:zip 字节 + 失败明细(部分失败仍返回 partial 包,调用方决定是否丢弃)。
func BuildBytes(req ExportRequest, provider CanonicalProvider) ([]byte, []string, error) {
	if len(req.Skills) == 0 {
		return nil, nil, ErrEmptySkills
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	now := time.Now().UTC().Format(time.RFC3339)
	manifest := PackageManifest{
		PkgFormat:  "skillbox.v1",
		CreatedAt:  now,
		SourceApp:  req.SourceApp,
		SourceDesc: req.SourceDesc,
		Skills:     []SkillSummary{},
	}
	failures := []string{}

	// 写每个 skill
	for _, ref := range req.Skills {
		c, ok, err := provider.LoadCanonical(ref.Scope, ref.ProjectID, ref.Name, ref.Version)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s@%s: %v", ref.Name, ref.Version, err))
			continue
		}
		if !ok {
			failures = append(failures, fmt.Sprintf("%s@%s: not found in store", ref.Name, ref.Version))
			continue
		}

		key := fmt.Sprintf("%s@%s", c.Manifest.Name, c.Manifest.Version)
		dir := fmt.Sprintf("skills/%s/", key)

		// 1) skill.yaml(manifest)
		if err := writeManifestYAML(w, dir+"skill.yaml", c.Manifest); err != nil {
			failures = append(failures, fmt.Sprintf("%s: write manifest: %v", key, err))
			continue
		}
		// 2) 全部 file
		for _, f := range c.Files {
			if err := writeFile(w, dir+f.Path, []byte(f.Content)); err != nil {
				failures = append(failures, fmt.Sprintf("%s/%s: %v", key, f.Path, err))
			}
		}
		// 3) 加索引
		manifest.Skills = append(manifest.Skills, SkillSummary{
			Key:       key,
			Name:      c.Manifest.Name,
			Version:   c.Manifest.Version,
			Scope:     ref.Scope,
			ProjectID: ref.ProjectID,
			Author:    c.Manifest.Author,
			License:   c.Manifest.License,
		})
	}

	// 写包级 manifest
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, failures, fmt.Errorf("marshal package manifest: %w", err)
	}
	if err := writeFile(w, manifestFile, manifestBytes); err != nil {
		return nil, failures, fmt.Errorf("write package manifest: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, failures, fmt.Errorf("close zip writer: %w", err)
	}

	if len(manifest.Skills) == 0 {
		// 全部失败
		return nil, failures, fmt.Errorf("all skills failed: %v", failures)
	}
	return buf.Bytes(), failures, nil
}

// writeFile 把字节写成一个 zip entry;空路径直接 skip。
func writeFile(w *zip.Writer, name string, data []byte) error {
	fw, err := w.Create(name)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, bytes.NewReader(data)); err != nil {
		return err
	}
	return nil
}

// writeManifestYAML 简化的 manifest 序列化:
//   - 数组字段:逗号分隔风格
//   - 字符串:原样(不做 yaml 转义,反正 value 都是普通字符)
func writeManifestYAML(w *zip.Writer, name string, m skilladapter.Manifest) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "name: %s\n", m.Name)
	fmt.Fprintf(&b, "version: %s\n", m.Version)
	fmt.Fprintf(&b, "description: %q\n", m.Description)
	if len(m.Triggers) > 0 {
		fmt.Fprintf(&b, "triggers:\n")
		for _, t := range m.Triggers {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}
	if m.Author != "" {
		fmt.Fprintf(&b, "author: %s\n", m.Author)
	}
	if m.License != "" {
		fmt.Fprintf(&b, "license: %s\n", m.License)
	}
	if len(m.DependsOn) > 0 {
		fmt.Fprintf(&b, "depends_on:\n")
		for _, t := range m.DependsOn {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}
	if len(m.TargetTools) > 0 {
		fmt.Fprintf(&b, "target_tools:\n")
		for _, t := range m.TargetTools {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}
	return writeFile(w, name, b.Bytes())
}
