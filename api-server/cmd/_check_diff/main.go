package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func main() {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".skill-box", "skills")
	repo, _ := git.PlainOpen(root)

	from, _ := repo.CommitObject(plumbing.NewHash("d537f52390d70e7dd95800d544b39612270fefe7"))
	to, _ := repo.CommitObject(plumbing.NewHash("097a7036666445b95dcba0cfa7b5fffaac1c116e"))
	fromTree, _ := from.Tree()
	toTree, _ := to.Tree()

	fmt.Println("fromTree entries:", len(fromTree.Entries))
	fmt.Println("toTree entries:", len(toTree.Entries))

	// Patch 各种用法
	patch, err := fromTree.Patch(toTree)
	fmt.Printf("Patch err=%v msg_len=%d\n", err, len(patch.Message()))
	for _, ps := range patch.PatchScenarios() {
		fmt.Println("  scenario:", ps)
	}
	fmt.Println("  stats:", patch.Stats())

	// 对每个 entry 单独试
	for i, e := range fromTree.Entries {
		if i > 3 {
			break
		}
		ft, err := fromTree.Tree(e.Hash)
		if err != nil {
			fmt.Printf("  from subtree %s err=%v\n", e.Name, err)
			continue
		}
		tt, err := toTree.Tree(e.Hash)
		if err != nil {
			fmt.Printf("  to subtree %s err=%v\n", e.Name, err)
		} else {
			fmt.Printf("  subtree %s from=%d to=%d\n", e.Name, len(ft.Entries), len(tt.Entries))
			subPatch, _ := ft.Patch(tt)
			fmt.Printf("    sub msg_len=%d\n", len(subPatch.Message()))
		}
	}
}
