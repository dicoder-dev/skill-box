# 07-03 bugfix skillstore.loadFromDir 漏设 SourceDir

## 现象
首页启用工具时 home 端 skill(例 `unit-test-gen`)报错:

```
claude: claude: symlink apply: empty source_dir for skill "unit-test-gen"
```

## 根因
apply 链路(`sskillapp.Service.Apply`)走 `sskill.Get(name)` →
`skillstore.Store.LoadByName(name)` → `Store.loadFromDir(dir)` 拿 `Canonical`,
然后传 `skillapp.Applier.ApplyOne` → `BaseAdapter.ApplyLink(c, targetDir)`。
`ApplyLink` 校验 `c.SourceDir == ""` 时直接返错。

但 `Store.loadFromDir`(`api-server/internal/skillstore/store.go:176`)原本
**根本没给 `c.SourceDir` 赋值** —— 之前 `BaseAdapter.readSkillDir` 写到
`SourceDir` 那一改动在 2026-07-03 已修(post_onboarding_import 兜底),
但 home 端 store 三条入口(Load / LoadByName / LoadByPath)漏了同款写入,
导致 home 里 market 拉下来存的 skill 走 symlink 模式启用必炸。

上次提交 `51d470f` 只动了 `readSkillDir` + `post_onboarding_import`,
**store 这条主入口漏了**。

## 修复
在 `store.loadFromDir` return 前追加:

```go
if real, err := filepath.EvalSymlinks(dir); err == nil {
    c.SourceDir = real
} else {
    c.SourceDir = dir
}
```

与 `skilladapter.BaseAdapter.readSkillDir` 行为一致,用 EvalSymlinks 解析
真实路径,避免 symlink 链上路径在 `~/.claude/skills` 与
`~/.agents/skills/xxx` 之间漂移。

## 测试
新增 `TestLoad_PopulatesSourceDir`(覆盖 Load / LoadByName / LoadByPath
三条入口 + 浅层根目录场景)和 `TestLoadByName_PopulatesSourceDir_Nested`
(覆盖嵌套分组场景 `aa/unit-test-gen`)。

`s/internal/skillstore/` + `skilladapter` + `skillapp` + `sskillapp`
全套测试通过。

## 提交
`83c1f3b fix(skillstore): 修复 loadFromDir 漏设 SourceDir 导致 symlink apply 报错`
已 push 至 origin/main。

## 相关
- 2026-07-03 同款坑的另一个入口:`conboarding/post_onboarding_import.a.go`,
  见上次提交 `51d470f`。
- 新增项目 memory `docs/agent/memory/project.md` 末条:
  `canonical 唯一来源字段约定(2026-07-03)`,列出所有产 Canonical 的入口,
  防后续再漏。
