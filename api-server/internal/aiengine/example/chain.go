// Package example 演示如何用 blades.Agent / flow.SequentialAgent 组合多个 preset。
//
// 本文件是 2026-07-14 切到 go-kratos/blades 后的"留下可参考"的样例,
// 不挂到任何 controller,只展示用法。业务侧将来想做
// "先审文档再翻译" / "先 safety check 再 find_duplicates" 这种多步流水线时,
// 可以照着这个改。
//
// 关键点:
//   - 复用 aiengine.AllPresets / RenderPreset:不用把 system 文本再写一遍
//   - 每个 preset 包成一个 Agent(blades.WithModel + WithInstructionProvider)
//   - flow.NewSequentialAgent 把多个 Agent 串成 Chain
//   - 输入是 SKILL.md 全文,Generator[*Message, error] 流式消费
package example

import (
	"context"
	"fmt"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/flow"
	"ginp-api/internal/aiengine"
)

// SafetyThenTranslate 演示"先用 safety_check preset 跑一次,再调 translate_skill"。
//
// 怎么用:
//
//	engine := aiengine.NewEngine(st)  // st 是 settings.Service
//	model, _, _ := engine.Build(&entity.AIProvider{Name: "openai", Kind: "openai", ...})
//	agent, _ := SafetyThenTranslate(model)
//	inv := blades.NewInvocation(blades.UserMessage(skillMDContent), nil, nil)
//	for msg, err := range agent.Run(ctx, inv) { ... }
//
// 注:这个 Chain 目前只是把两个 agent 的输出原样流式传出;
// 要做"if safety pass then translate"这种条件分支,用 flow.NewRoutingAgent。
func SafetyThenTranslate(model blades.ModelProvider) (blades.Agent, error) {
	safetyAgent, err := blades.NewAgent(
		"safety_check",
		blades.WithModel(model),
		blades.WithDescription("扫 SKILL.md 的安全/合规问题"),
		blades.WithInstructionProvider(func(ctx context.Context) (string, error) {
			return findPresetSystem("safety_check"), nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("new safety agent: %w", err)
	}

	translateAgent, err := blades.NewAgent(
		"translate_skill",
		blades.WithModel(model),
		blades.WithDescription("把 SKILL.md 翻译到目标语言"),
		blades.WithInstructionProvider(func(ctx context.Context) (string, error) {
			return findPresetSystem("translate_skill"), nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("new translate agent: %w", err)
	}

	chain := sequential("safety_then_translate", "先安全审计,再翻译", safetyAgent, translateAgent)
	return chain, nil
}

// sequential 包一层 flow.NewSequentialAgent,避免业务侧重复 import flow 包。
func sequential(name, desc string, subs ...blades.Agent) blades.Agent {
	return flow.NewSequentialAgent(flow.SequentialConfig{
		Name:        name,
		Description: desc,
		SubAgents:   subs,
	})
}

// findPresetSystem 拿 preset 的 system 文本。blades 内部没存这个,业务侧再读一次 aiengine.AllPresets。
func findPresetSystem(id string) string {
	for _, p := range aiengine.AllPresets {
		if p.ID == id {
			return p.System
		}
	}
	return ""
}
