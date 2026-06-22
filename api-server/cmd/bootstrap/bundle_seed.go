package bootstrap

import (
	"log"
	"strings"

	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillbundle"
	"ginp-api/internal/skillstore"

	mskill "ginp-api/internal/gapi/model/skillbox/mskill"
)

// BundleSource 标记 bundle 写入的 source;前端可用这个区分"内置 vs 用户自建"。
const BundleSource = "bundle"

// SeedBundledSkills 把 embed 进去的 6 个预置 skill seed 到磁盘 + DB。
//
// 行为:
//   - 已存在同 (scope=global, name, version) 时跳过(幂等)
//   - 启动失败不 panic:DB 写入错误只 log;store 写盘错误也只 log
//   - 不阻塞后续服务启动(本函数由 Boot 在 StartDB 之后调,seed 完成前
//     HTTP server 还没起,所以最坏情况只是日志报几条 error)
//
// 为什么在 bootstrap 而不是 service:启动期一次性数据,与 HTTP 请求生命周期无关;
// 放到 cmd 层让 gapi / web / desktop 三处入口共享同一份 seed 逻辑。
func SeedBundledSkills() {
	dbWrite := dbs.GetWriteDb()
	if dbWrite == nil {
		log.Printf("seed: write db unavailable, skip")
		return
	}

	store, err := buildStoreForSeed()
	if err != nil {
		log.Printf("seed: build store failed: %v", err)
		return
	}
	svc := sskill.New(dbWrite, dbWrite, store)

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

// seedOne 单个 skill 的 seed:写入 store(物理目录) → 检查 DB 是否已有 → 没有就 Create。
//
// 顺序:store 先,DB 后。store 失败立即返回(没物理文件就别登记元数据);
// DB 失败要回滚 store(避免孤儿目录),由 sskill.Create 自己负责。
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

	_, err = svc.Create(&sskill.WriteInput{
		Scope:     skilladapter.ScopeGlobal,
		ProjectID: 0,
		Name:      c.Manifest.Name,
		Version:   version,
		Source:    BundleSource,
		SourceRef: "skillbox/builtin",
		Manifest:  c.Manifest,
		Files:     c.Files,
	})
	return err
}

// findSkillByName 按 name 查一条(global scope,project_id=0);只用于幂等检查。
func findSkillByName(svc *sskill.Service, name string) (bool, error) {
	res, err := svc.List(sskill.ListQuery{Scope: skilladapter.ScopeGlobal, Keyword: name, Page: 1, Size: 50})
	if err != nil {
		return false, err
	}
	for _, it := range res.Items {
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
	// sskill 返回的 ErrStoreSave / DB unique 冲突都包含"already"语义
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "idx_skill_scope_proj_name")
}

// buildStoreForSeed 装配 Store;桌面端 StoreRoot 由 configs 注入,其它走默认 ~/.skillbox/store。
func buildStoreForSeed() (*skillstore.Store, error) {
	if root := strings.TrimSpace(configs.Skillbox.StoreRoot); root != "" {
		return skillstore.NewAt(root)
	}
	return skillstore.New()
}

// 编译期引用(防止 mskill 包被未使用警告,且为后续按 ID 查重留口)。
var _ = mskill.FieldID