// Package sskillpkg 提供 .skillbox 导出 / 导入的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.5 节):
//   - Export 走 skillpkg.BuildBytes(provider) — provider 是本包实现的 adapter,复用 sskill.GetFull
//   - Import 走 skillpkg.Importer(provider) — provider 复用 sskill.Service.Create
//   - 上传/下载:Export 直接返 []byte 给 controller,Import 从 controller 拿 []byte
package sskillpkg

import (
	"errors"
	"fmt"

	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillpkg"
)

// 业务错误。
var (
	ErrInvalidScope = errors.New("skillpkg: target scope must be 'global' or 'project'")
)

// Service 业务服务。
type Service struct {
	skillSvcFactory func() (*sskill.Service, error)
}

func New(skillSvcFactory func() (*sskill.Service, error)) *Service {
	return &Service{skillSvcFactory: skillSvcFactory}
}

// sskillAdapter 把 sskill.Service 适配成 skillpkg.CanonicalProvider / SkillInstaller。
type sskillAdapter struct {
	svc *sskill.Service
}

func (a *sskillAdapter) LoadCanonical(scope string, projectID uint, name, version string) (skilladapter.Canonical, bool, error) {
	full, err := a.svc.GetFull(scope, name, version, projectID)
	if err != nil {
		if errors.Is(err, sskill.ErrNotFound) {
			return skilladapter.Canonical{}, false, nil
		}
		return skilladapter.Canonical{}, false, err
	}
	return full.Canonical, true, nil
}

func (a *sskillAdapter) InstallCanonical(scope string, projectID uint, c skilladapter.Canonical, source string) (uint, error) {
	in := &sskill.WriteInput{
		Scope:     scope,
		ProjectID: projectID,
		Name:      c.Manifest.Name,
		Version:   c.Manifest.Version,
		Source:    "imported",
		SourceRef: source,
		Manifest:  c.Manifest,
		Files:     c.Files,
	}
	row, err := a.svc.Create(in)
	if err != nil {
		return 0, fmt.Errorf("skillpkg: install: %w", err)
	}
	return row.ID, nil
}

// BuildExport 业务层入口:返回一个 (bytes, failures, error)。
// caller(controller)负责把 bytes 写到 HTTP 响应。
func (s *Service) BuildExport(req skillpkg.ExportRequest) ([]byte, []string, error) {
	svc, err := s.skillSvcFactory()
	if err != nil {
		return nil, nil, fmt.Errorf("skillpkg: skillSvcFactory: %w", err)
	}
	provider := &sskillAdapter{svc: svc}
	return skillpkg.BuildBytes(req, provider)
}

// ParseManifest 业务层入口:解析 zip 字节流拿 manifest(用于前端预览包内容)。
func (s *Service) ParseManifest(zipBytes []byte) (*skillpkg.PackageManifest, error) {
	return skillpkg.ParseManifest(zipBytes)
}

// Import 业务层入口:把 zip 装入 store。
func (s *Service) Import(zipBytes []byte, req skillpkg.ImportRequest) (*skillpkg.ImportResult, error) {
	if req.TargetScope != skilladapter.ScopeGlobal && req.TargetScope != skilladapter.ScopeProject {
		return nil, ErrInvalidScope
	}
	if req.TargetScope == skilladapter.ScopeProject && req.ProjectID == 0 {
		return nil, fmt.Errorf("skillpkg: project_id required when target_scope=project")
	}
	svc, err := s.skillSvcFactory()
	if err != nil {
		return nil, fmt.Errorf("skillpkg: skillSvcFactory: %w", err)
	}
	inst := skillpkg.NewImporter(&sskillAdapter{svc: svc})
	return inst.Install(zipBytes, req)
}
