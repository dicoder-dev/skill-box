package skillversion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ginp-api/configs"
)

// BootstrapInit 启动时一次性兜底初始化(2026-07-17 增,供 cmd/bootstrap 调用)。
//
// 流程:
//  1. 若 ~/.skill-box/skills/.git/ 不存在 → PlainInit
//  2. 扫描 root 下所有 <group>/<name>/SKILL.md 子目录,若 git log 是空(只有
//     InitIfNotExists 的"initialize empty repository"commit),把所有现有 skill
//     文件入库成 "chore(skills): initial import" 一个 commit
//  3. 若已 init + 已有 commit → 什么都不做(交给 store.Save hook 持续增量)
//
// 2026-07-17 设计:
//   - 不强依赖 store.Save 触发首次 commit(用户首次启动后没新建 skill 就退,
//     git 仓库永远只有空 init commit,毫无意义)
//   - 不重复 commit:扫描 worktree status,IsClean() 时直接 skip
//   - 失败仅 stderr,不阻塞 bootstrap
func BootstrapInit() {
	defer func() {
		_ = recover()
	}()

	repo, err := Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[skillversion] bootstrap init: %v\n", err)
		return
	}

	if !repo.IsInitialized() {
		if err := repo.InitIfNotExists(); err != nil {
			fmt.Fprintf(os.Stderr, "[skillversion] init: %v\n", err)
			return
		}
	}

	// 检查 worktree status;若脏 → 说明 store.Save 已经在干活,跳过
	wstatus, werr := repo.worktreeStatus()
	if werr != nil {
		fmt.Fprintf(os.Stderr, "[skillversion] worktree status: %v\n", werr)
		return
	}
	if !wstatus {
		return
	}

	// 扫描所有现有 skill,做一次兜底入库
	root := repo.Root()
	if root == "" {
		root = strings.TrimSpace(configs.Skillbox.StoreRoot)
	}
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".skill-box", "skills")
	}
	skills := scanSkillsForImport(root)
	if len(skills) == 0 {
		return
	}
	msg := fmt.Sprintf("chore(skills): initial import (%d skills)", len(skills))
	_, err = repo.AutoCommitAndPush(CommitInput{
		Message: msg,
		Paths:   nil, // add all
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[skillversion] bootstrap commit: %v\n", err)
	}
}

// worktreeStatus 内部 helper:返回 worktree 是否有未提交改动。
func (r *Repo) worktreeStatus() (bool, error) {
	repository, err := r.open()
	if err != nil {
		return false, err
	}
	wt, err := repository.Worktree()
	if err != nil {
		return false, err
	}
	st, err := wt.Status()
	if err != nil {
		return false, err
	}
	return !st.IsClean(), nil
}

// scanSkillsForImport 扫描 root 下所有 <group>/<name>/SKILL.md 子目录,
// 返每个 skill 的相对路径列表(用于 commit message 注释,实际 commit 走 add all)。
func scanSkillsForImport(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(p) == "SKILL.md" {
			rel, rerr := filepath.Rel(root, filepath.Dir(p))
			if rerr == nil && rel != "." {
				out = append(out, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	return out
}

// FlushPushQueue 进程退出前刷盘 push_queue(供 cmd/bootstrap.RegisterExitHook 调用)。
func FlushPushQueue() {
	defer func() { _ = recover() }()
	pushQueue.Flush()
}