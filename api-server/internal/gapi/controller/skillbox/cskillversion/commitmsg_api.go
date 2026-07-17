// 自动 commit message 设置 + LLM 测试 HTTP 端点(2026-07-18 增)。
//
// 三个端点:
//   GET  /api/skillbox/git/auto-commit      读 LLMEnabled + Template + LLM 可用性
//   POST /api/skillbox/git/auto-commit      写 LLMEnabled / Template
//   POST /api/skillbox/git/llm-test         现场跑一次最小 prompt 测 LLM(用于"测试"按钮)
package cskillversion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/configs"
	"ginp-api/internal/aiengine"
	"ginp-api/internal/commitmsg"
	"ginp-api/internal/db/dbs"
	maiprovider "ginp-api/internal/gapi/model/skillbox/maiprovider"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// ===========================================================================
// Request / Response
// ===========================================================================

// RespondAutoCommit GET 响应。
type RespondAutoCommit struct {
	LLMEnabled bool                 `json:"llm_enabled"`
	Template   string               `json:"template"`
	// Available=true 当至少一个 enabled provider 配了 api key。
	// 当 false 时,前端 UI 把 LLMEnabled checkbox 禁用 + 展示 Reason。
	Available bool                 `json:"llm_available"`
	Reason    string               `json:"reason,omitempty"`
	// ProviderName 给前端展示"已用 X (model Y)" 信息(不强制)。
	ProviderName string `json:"provider_name,omitempty"`
}

// RequestAutoCommit POST 入参。
//
// 2026-07-18 决策:
//   - LLMEnabled 写入前必须 server 端再次校验 LLM Available(防止前端绕 UI
//     直接发 true 同时 LLM 已挂) — 失败时返 400 + reason。
//   - Template 不强制枚举校验,落到 commitmsg 层会有默认值兜底。
type RequestAutoCommit struct {
	LLMEnabled *bool   `json:"llm_enabled"`
	Template   *string `json:"template"`
}

// ===========================================================================
// LLM 可用性聚合
// ===========================================================================

// autoCommitContext 聚合"判定 LLM 是否可用"所需的依赖。
//
// 2026-07-18 设计:
//   - db ai_providers 行 + settings(api key) + aiengine.Engine 构造 ModelProvider
//   - 本结构由 GetAutoCommitStatus / TestLLM / skillstore.autoCommitAfterSave 三个调用点共用
//   - 不放全局 singleton: skillstore 在 goroutine 里跑,直接 NewService 现场拼一遍
//     (cheap,ai_providers 行数小,settings 拿 key 走 KV 缓存)
type autoCommitContext struct {
	st        *settings.Service
	ai        *maiprovider.Model
	eng       *aiengine.Engine
	providers []*aiengine.Config
	// 选优先级最高的 enabled provider(返回 nil 当没有 enabled)
	picked *pickedProvider
}

type pickedProvider struct {
	cfg aiengine.Config
	row *struct {
		Name string
		Kind string
	}
}

// newAutoCommitContext 现场拼一组依赖,cheap(无 IO,只 DB 读 ai_providers)。
//
// 失败时返 nil + error — 调用方默认按 LLM 不可用处理。
func newAutoCommitContext() (*autoCommitContext, error) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	ai := maiprovider.NewModel(dbs.GetWriteDb(), dbs.GetReadDb())
	rows, _, err := ai.FindList(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("auto_commit: list providers: %w", err)
	}
	eng := aiengine.NewEngine(aiSecretStore(st))

	var enabled []*aiengine.Config
	var picked *pickedProvider
	for _, r := range rows {
		if !r.Enabled {
			continue
		}
		cfg := &aiengine.Config{Name: r.Name, Kind: r.Kind, BaseURL: r.BaseURL, Model: r.Model}
		enabled = append(enabled, cfg)
		key, _, _ := st.Get("ai:" + r.Name + ":api_key")
		if picked == nil && key != "" {
			picked = &pickedProvider{
				cfg: *cfg,
				row: &struct {
					Name string
					Kind string
				}{Name: r.Name, Kind: r.Kind},
			}
		}
	}
	return &autoCommitContext{
		st:        st,
		ai:        ai,
		eng:       eng,
		providers: enabled,
		picked:    picked,
	}, nil
}

// aiSecretStore 把 settings.Service 包装成 aiengine.SecretStore。
func aiSecretStore(st *settings.Service) aiengine.SecretStore {
	return aiSecretStoreImpl{st: st}
}

type aiSecretStoreImpl struct{ st *settings.Service }

func (a aiSecretStoreImpl) Resolve(name string) (string, error) {
	v, _, err := a.st.Get("ai:" + name + ":api_key")
	return v, err
}

// ===========================================================================
// Handlers
// ===========================================================================

// GetAutoCommitStatus GET /api/skillbox/git/auto-commit
func GetAutoCommitStatus(c *ginp.ContextPlus) {
	ctx, err := newAutoCommitContext()
	if err != nil {
		logger.Warn("auto-commit status: %v", err)
	}
	cfg := configs.Skillbox.AutoCommit
	resp := RespondAutoCommit{
		LLMEnabled: cfg.LLMEnabled,
		Template:   firstNonEmpty(cfg.Template, "filename"),
	}
	if ctx != nil {
		if ctx.picked != nil {
			resp.Available = true
			resp.ProviderName = ctx.picked.row.Name
		} else if len(ctx.providers) > 0 {
			resp.Available = false
			resp.Reason = "provider 已启用但未配置 API key"
		} else {
			resp.Available = false
			resp.Reason = "还没有可用的 AI provider"
		}
	} else {
		resp.Available = false
		resp.Reason = "无法读取 AI provider 列表"
	}
	c.JSON(200, resp)
}

// SaveAutoCommit POST /api/skillbox/git/auto-commit
//
// 校验:若用户发 LLMEnabled=true 但 server 侧判定不可用,返 400 让前端
// 知道(不强制改,只是提示)。
func SaveAutoCommit(c *ginp.ContextPlus, req *RequestAutoCommit) {
	if req.LLMEnabled != nil {
		if *req.LLMEnabled {
			ctx, _ := newAutoCommitContext()
			if ctx == nil || ctx.picked == nil {
				c.JSON(400, gin.H{
					"error":   "LLM 暂不可用,请先在 AI 设置里配置可用 provider + API key,并通过测试",
					"reason":  "unavailable",
				})
				return
			}
		}
		configs.Skillbox.AutoCommit.LLMEnabled = *req.LLMEnabled
	}
	if req.Template != nil {
		t := strings.TrimSpace(*req.Template)
		if t == "" {
			t = "filename"
		}
		switch commitmsg.Template(t) {
		case commitmsg.TemplateTimestamp, commitmsg.TemplateFilename, commitmsg.TemplateOpFiles:
			configs.Skillbox.AutoCommit.Template = t
		default:
			c.JSON(400, gin.H{"error": "unknown template: " + t})
			return
		}
	}
	c.JSON(200, gin.H{"ok": true})
}

// TestLLM POST /api/skillbox/git/llm-test
//
// 现场跑一次最简 prompt,5s 内返结果;用于"测试 LLM"按钮。
// 返 200 + {ok, model, output} 或 200 + {ok:false, reason}。
func TestLLM(c *ginp.ContextPlus) {
	ctx, err := newAutoCommitContext()
	if err != nil {
		c.JSON(200, gin.H{"ok": false, "reason": err.Error()})
		return
	}
	if ctx.picked == nil {
		c.JSON(200, gin.H{"ok": false, "reason": "no enabled provider or api key missing"})
		return
	}
	prov, err := ctx.eng.BuildFromConfig(ctx.picked.cfg, mustSecret(ctx.st, ctx.picked.cfg.Name))
	if err != nil {
		c.JSON(200, gin.H{"ok": false, "reason": "build provider: " + err.Error()})
		return
	}

	cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ccancel()
	stream := prov.NewStreaming(cctx, newLLMStreamReq("Reply with one word: pong"))
	var out strings.Builder
	for resp, err := range stream {
		if err != nil {
			c.JSON(200, gin.H{"ok": false, "reason": "stream: " + err.Error()})
			return
		}
		if resp == nil || resp.Message == nil {
			continue
		}
		out.WriteString(resp.Message.Text())
	}
	if cctx.Err() != nil {
		c.JSON(200, gin.H{"ok": false, "reason": "timeout"})
		return
	}
	c.JSON(200, gin.H{
		"ok":     true,
		"model":  ctx.picked.cfg.Model,
		"output": strings.TrimSpace(out.String()),
	})
}

// ===========================================================================
// helpers(本文件内私有)
// ===========================================================================

// ----------------------------------------------------------------------------
// 2026-07-18 增:LLM 生成器全局注册 — skillstore 在 autoCommitAfterSave goroutine
// 里通过 commitmsg.Generate(ctx, options) 拿消息,不能反向 import controller,
// 所以这里把 BuildCommitLLMSender 注册到 commitmsg 全局。
//
// 不放 init() — 在 NewEngine / first call 时再 lazy 注册,避免 controller 没
// 装好时调用。但本端目前只有一个入口,直接 init() 简单。
// ----------------------------------------------------------------------------

func init() {
	// 2026-07-18 fix:不要在 init() 阶段立即调 BuildCommitLLMSender() —
	// 它内部 newAutoCommitContext() 会走 dbs.GetWriteDb(),彼时 InitDb
	// 还没跑,直接 panic("数据库未初始化")。
	//
	// 注册一个"现场拼"的外层闭包:每次 commitmsg.Generate 真要调 LLM
	// 时才 BuildCommitLLMSender,DB 已就绪;DB 仍不可用就返 nil 让
	// commitmsg 走模板路径。生成器本身 cheap,不持有 conn。
	commitmsg.SetGlobalLLMGenerator(func(ctx context.Context, prompt string) (string, error) {
		sender := BuildCommitLLMSender()
		if sender == nil {
			return "", errors.New("auto_commit: llm sender unavailable")
		}
		return sender(ctx, prompt)
	})
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) == "" {
		return b
	}
	return a
}

func mustSecret(st *settings.Service, name string) string {
	v, _, _ := st.Get("ai:" + name + ":api_key")
	return v
}

// BuildCommitLLMSender 是 commitmsg.LLMGenerate 的 aiengine 实现(给 skillstore 调用)。
//
// 2026-07-18:cskillversion.init() 通过 commitmsg.SetGlobalLLMGenerator 把本函数
// 注册到 commitmsg 全局;skillstore 在 autoCommitAfterSave goroutine 里调
// commitmsg.Generate 时,如果 opts.LLM==nil 会回退到此函数 — 让 store 包不必
// 反向 import gapi/controller。
//
// 行为:
//   - 现场拼 autoCommitContext,选第一个 enabled provider(>api_key 非空)。
//   - 不可用时返 nil — commitmsg.Generate 收到 nil 走模板路径。
//   - 每次调用重新构建 ModelProvider(cheap,不持有 Conn)。
func BuildCommitLLMSender() commitmsg.LLMGenerate {
	ctx, err := newAutoCommitContext()
	if err != nil || ctx == nil {
		return nil
	}
	return ctx.buildLLMSender()
}

// buildLLMSender 是 ctx 上的方法,内部用 ctx 的依赖直接闭包。
func (ctx *autoCommitContext) buildLLMSender() commitmsg.LLMGenerate {
	if ctx == nil || ctx.picked == nil || ctx.eng == nil {
		return nil
	}
	cfg := ctx.picked.cfg
	st := ctx.st
	eng := ctx.eng
	return func(parent context.Context, prompt string) (string, error) {
		key, err := aiSecretStore(st).Resolve(cfg.Name)
		if err != nil || key == "" {
			return "", errors.New("api key missing")
		}
		prov, err := eng.BuildFromConfig(cfg, key)
		if err != nil {
			return "", err
		}
		req := newLLMStreamReq(prompt)
		stream := prov.NewStreaming(parent, req)
		var out strings.Builder
		for resp, err := range stream {
			if err != nil {
				return "", err
			}
			if resp == nil || resp.Message == nil {
				continue
			}
			out.WriteString(resp.Message.Text())
		}
		if cerr := parent.Err(); cerr != nil {
			return "", cerr
		}
		return out.String(), nil
	}
}
