package skilltester

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/blades"
	"ginp-api/internal/aiengine"
	"ginp-api/internal/skilladapter"
)

// AI 走查:走 aiengine + 内置 preset。Provider 注入由 caller 负责。
//
// 降级策略:
//   - 没有可用 provider -> skipped
//   - provider 没配 key -> skipped
//   - preset 渲染失败 -> errored
//   - 实际流式对话失败 -> errored
//   - 流跑完且收到 done -> 把完整文本存 detail,passed

// AISummary 详情。
type AISummary struct {
	Preset   string `json:"preset"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Output   string `json:"output,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// AIWalker 注入 aiengine.Config 列表 + Secret + Build 闭包。
// Build 返回的是 blades.ModelProvider(2026-07-14 切到 blades 后);
// skilltester 不再依赖 aiengine.Provider 接口。
type AIWalker struct {
	// Providers 是 ai_providers 表的当前快照,由 service 层查询后传入。
	Providers []*aiengine.Config
	// Secret 凭据解析(provider_name -> api_key)。
	Secret func(name string) (string, error)
	// Build 拿到 Config 返回 blades.ModelProvider;func 由 service 层闭包注入(避免 skilltester 反向依赖 sai)。
	Build func(cfg aiengine.Config) (blades.ModelProvider, error)
}

// RunAIWalk 默认走 safety_check preset。无 provider / 无 key 时 skipped。
func RunAIWalk(c skilladapter.Canonical, walker *AIWalker, opts Options) CheckResult {
	if walker == nil {
		summary := AISummary{Skipped: true, Reason: "no ai walker configured"}
		b, _ := json.Marshal(summary)
		return CheckResult{Check: CheckAI, Status: StatusSkipped, Message: summary.Reason, Detail: string(b)}
	}
	presetID := opts.AIPreset
	if presetID == "" {
		presetID = "safety_check"
	}
	preset, ok := findPreset(presetID)
	if !ok {
		return CheckResult{Check: CheckAI, Status: StatusErrored, Message: "unknown preset " + presetID}
	}

	// 选 provider(优先按 opts.AIProvider,否则按 priority 选)
	cfg, err := pickProvider(walker.Providers, opts.AIProvider)
	if err != nil {
		summary := AISummary{Preset: presetID, Skipped: true, Reason: err.Error()}
		b, _ := json.Marshal(summary)
		return CheckResult{Check: CheckAI, Status: StatusSkipped, Message: err.Error(), Detail: string(b)}
	}

	prov, err := walker.Build(cfg)
	if err != nil {
		return CheckResult{Check: CheckAI, Status: StatusErrored, Message: "build provider: " + err.Error()}
	}
	key, err := walker.Secret(cfg.Name)
	if err != nil || key == "" {
		summary := AISummary{Preset: presetID, Provider: cfg.Name, Skipped: true, Reason: "no api key configured"}
		b, _ := json.Marshal(summary)
		return CheckResult{Check: CheckAI, Status: StatusSkipped, Message: summary.Reason, Detail: string(b)}
	}

	// 拼 skill 全文(把所有 file 拼成 markdown)
	skillMD := buildSkillMDForPrompt(c)
	messages := aiengine.RenderPreset(preset, map[string]string{"skill_md": skillMD})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 构造 ModelRequest。apiKey 在 model 实例化时已注入,这里只传 messages。
	req := &blades.ModelRequest{Messages: messages}
	stream := prov.NewStreaming(ctx, req)

	var (
		out      strings.Builder
		timedOut bool
	)
	for resp, err := range stream {
		if err != nil {
			summary := AISummary{Preset: presetID, Provider: cfg.Name, Model: cfg.Model, Reason: err.Error()}
			b, _ := json.Marshal(summary)
			return CheckResult{Check: CheckAI, Status: StatusErrored, Message: "ai error: " + err.Error(), Detail: string(b)}
		}
		if resp == nil || resp.Message == nil {
			continue
		}
		if text := resp.Message.Text(); text != "" {
			out.WriteString(text)
		}
	}
	if ctx.Err() != nil {
		timedOut = true
	}

	summary := AISummary{Preset: presetID, Provider: cfg.Name, Model: cfg.Model, Output: out.String()}
	b, _ := json.Marshal(summary)
	res := CheckResult{Check: CheckAI, Detail: string(b)}
	switch {
	case timedOut:
		res.Status = StatusErrored
		res.Message = "ai walkthrough timeout"
	case out.Len() == 0:
		// 没产出但没报错:也算 skipped(避免被当 failed 吓到用户)
		res.Status = StatusSkipped
		res.Message = "ai returned empty output"
	default:
		res.Status = StatusPassed
		res.Message = "ai walkthrough completed"
	}
	return res
}

func findPreset(id string) (aiengine.Preset, bool) {
	for _, p := range aiengine.AllPresets {
		if p.ID == id {
			return p, true
		}
	}
	return aiengine.Preset{}, false
}

// pickProvider 简单选:显式指定则按 name;否则按 cfg.Model 不空 + 任意 priority 选第一个。
// (优先级排序在 service 层完成,这里只消费结果)
func pickProvider(providers []*aiengine.Config, name string) (aiengine.Config, error) {
	if len(providers) == 0 {
		return aiengine.Config{}, errors.New("no ai provider enabled")
	}
	if name != "" {
		for _, p := range providers {
			if p.Name == name {
				return *p, nil
			}
		}
		return aiengine.Config{}, errors.New("ai provider " + name + " not found or disabled")
	}
	return *providers[0], nil
}

// buildSkillMDForPrompt 把 canonical 拼成一段 markdown 文本喂给 AI。
func buildSkillMDForPrompt(c skilladapter.Canonical) string {
	var b strings.Builder
	// 复用 RenderSkillMD 的逻辑,但直接手写更可控
	if c.Manifest.Name != "" {
		fmt.Fprintf(&b, "name: %s\n", c.Manifest.Name)
	}
	if c.Manifest.Version != "" {
		fmt.Fprintf(&b, "version: %s\n", c.Manifest.Version)
	}
	if c.Manifest.Description != "" {
		fmt.Fprintf(&b, "description: %s\n", c.Manifest.Description)
	}
	if len(c.Manifest.Triggers) > 0 {
		fmt.Fprintf(&b, "triggers: [%s]\n", strings.Join(c.Manifest.Triggers, ", "))
	}
	b.WriteString("\n---\n\n")
	for _, f := range c.Files {
		fmt.Fprintf(&b, "## File: %s\n```\n%s\n```\n\n", f.Path, f.Content)
	}
	return b.String()
}
