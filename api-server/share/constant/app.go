// Package constant 集中维护跨模块的固定值。
//
// 命名约定:业务默认值(应用名、目录名等)统一放这里,避免散落到 configs /
// 各处 init() 里。任何需要"绝对路径"或"目录名"的代码都从这里读。
//
// 任何带 I/O 或派生的逻辑函数都不要放本包,放到 share/func 下,保持本包
// 永远是纯常量声明。
package constant

// AppName 应用目录名(~/.<AppName>/)。
//
// 同时作为 sqlite 文件名、配置文件名、logs/ 子目录的统一前缀。
// 改值要慎重 — 已有数据目录会被遗留,不会自动迁移。
//
// 业务侧展示名仍由 configs.System.AppName 控制(允许不同,如"dianji"
// 是项目代号,展示名可以叫别的)。
const AppName = "skill-box"
