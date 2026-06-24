package bootstrap

import (
	"log"
	"strings"

	"ginp-api/configs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillbundle"
	"ginp-api/internal/skillstore"
)

// BundleSource 标记 bundle 写入的 source;前端可用这个区分"内置 vs 用户自建"。
const BundleSource = "bundle"

// SeedBundledSkills 把 embed 进去的预置 skill seed 到磁盘。
//
// 2026-06-24 改造:skill 不再落 DB(MSkill 表已弃用),seed 只负责把 SKILL.md
// 写到 ~/.skill-box/skills/<name>/。幂等:目录已存在就跳过。
//
// 行为:
//   - store.Save 覆盖式;遇到"已存在"语义错误跳过
//   - 启动失败不 panic:只 log
//   - 不阻塞后续服务启动(本函数由 Boot 在 StartDB 之后调,seed 完成前
//     HTTP server 还没起,所以最坏情况只是日志报几条 error)
//
// 为什么在 bootstrap 而不是 service:启动期一次性数据,与 HTTP 请求生命周期无关;
// 放到 cmd 层让 gapi / web / desktop 三处入口共享同一份 seed 逻辑。
func SeedBundledSkills() {
	store, err := buildStoreForSeed()
	if err != nil {
		log.Printf("seed: build store failed: %v", err)
		return
	}
	svc := sskill.New(store)

	canonicals, err := skillbundle.LoadAll()
	if err != nil {
		log.Printf("seed: load bundled skills failed: %v", err)
		return
	}

	skipped, inserted, failed := 0, 0, 0
	for _, c := range canonicals {
		if err := seedOne(svc, c); err != nil {
			if isAlreadyExists(err) {
				skipped++
				continue
			}
			log.Printf("seed: %s failed: %v", c.Manifest.Name, err)
			failed++
			continue
		}
		inserted++
	}
	log.Printf("seed: bundled skills done — inserted=%d skipped=%d failed=%d", inserted, skipped, failed)
}

// seedOne 单个 skill 的 seed。
//
// 顺序:先看 store 是否已有同名 skill → 有就 errAlreadyExists;没有就 Create(写盘)。
// source/source_ref 信息写到 Manifest 字段,不再单独传。
func seedOne(svc *sskill.Service, c skilladapter.Canonical) error {
	exists, err := findSkillByName(svc, c.Manifest.Name)
	if err != nil {
		return err
	}
	if exists {
		return errAlreadyExists
	}

	version := c.Manifest.Version
	if strings.TrimSpace(version) == "" {
		version = "0.1.0"
	}
	// 把 source 信息写进 manifest(2026-06-24:Source/SourceRef 不再单独入参)
	c.Manifest.Source = BundleSource
	c.Manifest.SourceRef = "skillbox/builtin"
	_, err = svc.Create(&sskill.WriteInput{
		Scope:    skilladapter.ScopeGlobal,
		Name:     c.Manifest.Name,
		Version:  version,
		Manifest: c.Manifest,
		Files:    c.Files,
	})
	return err
}

// findSkillByName 按 name 查一条(global scope);只用于幂等检查。
func findSkillByName(svc *sskill.Service, name string) (bool, error) {
	items, err := svc.List(name)
	if err != nil {
		return false, err
	}
	for _, it := range items {
		if it.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// errAlreadyExists 内部哨兵;不让 sskill 创建同名 skill 但又静默成功。
var errAlreadyExists = &seedError{msg: "already exists"}

type seedError struct{ msg string }

func (e *seedError) Error() string { return e.msg }

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	// sskill 返回的 ErrStoreSave / unique 冲突都包含"already"语义
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate")
}

// buildStoreForSeed 装配 Store;桌面端 StoreRoot 由 configs 注入,其它走默认 ~/.skill-box/skills。
func buildStoreForSeed() (*skillstore.Store, error) {
	if root := strings.TrimSpace(configs.Skillbox.StoreRoot); root != "" {
		return skillstore.NewAt(root)
	}
	return skillstore.New()
}
