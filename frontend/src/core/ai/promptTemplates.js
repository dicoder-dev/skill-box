// AI prompt 模板常量
//
// 2026-07-17 新建:把这几段模板从 i18n 字典里挪出来,根除 vue-i18n 解析报错。
//
// 原因:vue-i18n@9 在 legacy:false 模式下会用 @formatjs/intl-messageformat
// 解析所有消息字符串,当字符串里含 `"needs_apply":` 这种字面 JSON 时,解析器
// 会把 `{` 误识别成 ICU 占位符起始,抛出 "Invalid token in placeholder" 错误。
// 该错误会被 SkillFileInlinePanel.vue 的 onErrorCaptured 捕获,误显示成
// "技能详情加载出错"。
//
// 这些字符串不是用户可见文案,是发给 LLM 的 system prompt / 模板指令,
// 不需要 i18n,挪成纯 JS 常量最干净。

// 翻译:把当前 Markdown 文档翻译成目标语言。
// 占位符:
//   {target_lang}  — 目标语言的人类可读标签(由 AIRightPanel.langLabelOf 注入)
//   {skill_md}     — 当前文档全文
export const TRANSLATE_PROMPT_TEMPLATE =
`You are a Skill file translator. Translate the following Markdown document into {target_lang}.

Rules:
1) Keep frontmatter field names in English.
2) Code blocks, shell commands and file names must stay as-is.
3) Keep Markdown structure (headings, lists, links) — only translate the human text.
4) No preface, no explanation.

Document to translate:
\`\`\`
{skill_md}
\`\`\`

# Output format (strict)
Return exactly one JSON code block:
\`\`\`json
{{
  "needs_apply": true,
  "content": "the full translated markdown document (including frontmatter)",
  "reason": "Translated the full document into {target_lang}"
}}
\`\`\`
- needs_apply MUST be true (the translation replaces the original file).
- content is the complete translated markdown document.
- Output nothing outside the JSON code block.`

// 检测:审阅当前 Markdown 文档,给出问题清单。
// 占位符:
//   {skill_md}  — 当前文档全文
export const REVIEW_PROMPT_TEMPLATE =
`You are a Skill file reviewer. Carefully review the following Markdown document (Claude / Codex Skill) and list problems across these dimensions:
1) Grammar / spelling errors
2) Ambiguous or unclear wording
3) Inconsistencies with the frontmatter (description / triggers)
4) Missing or redundant sections
5) Deviations from Claude / Codex Skill best practices

Document to review:
\`\`\`
{skill_md}
\`\`\`

# Output format (strict)
Return exactly one JSON code block:
\`\`\`json
{{
  "needs_apply": false,
  "content": "",
  "reason": "Review findings:\\n- **Section X**: issue description\\n- **Section Y**: issue description"
}}
\`\`\`
- needs_apply MUST be false (a review is advisory, not a replacement).
- content is an empty string.
- reason contains the full Markdown bullet list of issues (use \\n for newlines); if nothing is wrong say "No obvious issues found." No greetings.`

// 自定义输入:用户直接在输入框里打字时,作为 system hint 拼到 user 消息前面,
// 让 LLM 自行判断 needs_apply 并返回严格 JSON。
// 该模板不含任何占位符。
export const CUSTOM_PROMPT_HINT =
`When the user's request is to translate / rewrite / optimize / complete / fix the file:
- needs_apply: true
- content: the full modified file content (to replace the original)
When the user is just chatting / asking a question / requesting a review:
- needs_apply: false
- content: empty string
- reason: a detailed answer to the user's question

# Output format (strict)
Return exactly one \`\`\`json code block:
\`\`\`json
{"needs_apply": boolean, "content": "string", "reason": "string"}
\`\`\`
Booleans use true / false (not the strings "true" / "false"). Output nothing outside the JSON code block.`