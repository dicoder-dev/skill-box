package logger

import "encoding/json"

// Event 把"业务审计事件"写到 INFO.csv(与普通 Info 同通道),
// 字段:action / target_type / name / fields 序列化为 JSON,作为日志内容单列写入。
// 用法:logger.Event("apply", "skill", name, map[string]any{"tool": t, "scope": s, ...})
//
// grep 友好:日志内容固定前缀 "audit ",可以 `grep audit ~/.skill-box/logs/2026-07/INFO.csv`
// 一次性把所有业务事件挑出来,等价于审计列表;前端不再提供页面。
func Event(action, targetType, name string, fields map[string]any) {
	envelope := map[string]any{
		"action":      action,
		"target_type": targetType,
		"name":        name,
		"fields":      fields,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		// 业务事件的字段序列化失败属于 coding bug,降级为字符串提示,不要阻塞主流程
		Info("audit action=%s target_type=%s name=%s marshal_err=%v",
			action, targetType, name, err)
		return
	}
	Info("audit %s", string(payload))
}
