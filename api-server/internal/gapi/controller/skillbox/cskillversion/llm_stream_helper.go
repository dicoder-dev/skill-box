// commitmsg_api.go 用到的 ai/blade helper(单独文件避免在主文件里堆 import)。
package cskillversion

import (
	"github.com/go-kratos/blades"
)

// newLLMStreamReq 构造一个 system + user 两轮 messages 的 stream request。
//
// 2026-07-18 设计:用 blades.SystemMessage / blades.UserMessage 工厂(blades v0.5.0
// 的 New* alias 不导出,直接走低层工厂)。
func newLLMStreamReq(prompt string) *blades.ModelRequest {
	return &blades.ModelRequest{
		Messages: []*blades.Message{
			blades.SystemMessage("You generate conventional commit messages. Output ONLY the line, no markdown, no quotes."),
			blades.UserMessage(prompt),
		},
	}
}
