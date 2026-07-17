// 中文语言包 - 完整覆盖 5 个视图 + AIPanel + App 全局。
//
// 命名空间按页面分:
//   - app.*     App.vue(品牌 / nav / health / stats)
//   - common.*  通用(取消 / 创建 / 保存 / 关闭 / 上一页 / 下一页 / 全部 / 应用过滤 / 处理中… 等)
//   - skills.*  SkillsView + AIPanel + Tag/Diff/Test/Editor
//   - projects.* ProjectsView
//   - market.*  MarketView
//   - onboarding.* OnboardingView

const messages = {
  app: {
    // 2026-07-11 改:品牌名回归 'Skill-Box'(此前误改为 'Q Boss')。
    // 同时把主题切换文案挪到 i18n(供侧栏底部 tooltip 使用)。
    // 2026-07-11 v11 改:大写 'SKILL-BOX' + 字体本身 letter-spacing 拉开,
    // 配合 weight 800 + 渐变 + 微高光呈现品牌级艺术感。
    // 2026-07-12 改:首字母大写 'SkillBox',去掉横杠;保留 weight 800 +
    // 渐变 + 微高光,但 letter-spacing 收紧回 0.5px(无全大写不需要拉宽)。
    brand: 'SkillBox',
    closeSidebar: '关闭侧栏',
    openSidebar: '打开侧栏',
    nav: {
      skills: { label: '技能' },
      projects: { label: '项目' },
      // 2026-07-01 增:工具元数据管理
      tools: { label: '工具' },
      market: { label: '市场' },
      onboarding: { label: '导入技能' },
      settings: { label: '设置' },
    },
    backendOk: '后端已连接',
    backendDown: '后端断开',
    refreshStats: '刷新统计',
    toolsLabel: '工具',
    // 2026-07-11 增:顶栏 tooltip 用
    themeToggle: {
      toDark: '切换到暗黑模式',
      toLight: '切换到亮色模式',
    },
  },

  common: {
    cancel: '取消',
    create: '创建',
    save: '保存',
    close: '关闭',
    delete: '删除',
    edit: '编辑',
    apply: '应用',
    search: '搜索',
    refresh: '刷新',
    prev: '上一页',
    next: '下一页',
    all: '全部',
    applyFilter: '应用过滤',
    processing: '处理中…',
    none: '—',
    dash: '—',
    confirm: '确认',
    loading: '加载中…',
    retry: '重试',
    optional: '可选',
    count: '个',
    notConfigured: '未配置',
    // 2026-07-01 改:去 1000 上限后 total 数字不代表全网真数,误导。
    // 改为只显示"第 X / Y 页",Y 是 totalPages 由 size 推算出来(可能 = 实际页数 / ctx 超时后是个偏小的估计)。
    pageOf: '第 {page} / {total} 页 · 共 {count} 条',
    // 2026-07-01 增:市场专用分页 — 隐藏"共 N 条"误导数字,只显示页数。
    pageOfNoCount: '第 {page} / {total} 页',
    totalCount: '共 {count} 条',
    noData: '该作用域下还没有技能',
    noDataHint: '点右上角"新建"开始,或去首次配置从已装工具导入',
    deleted: '已删除 {name}',
    openFailed: '打开失败:{msg}',
  },

  skills: {
    title: '技能',
    subtitle: '浏览 / 编辑 / 测试 / 落工具 / 打 tag / 回滚。AI 侧栏一键改写 frontmatter 与 body。',
    scopeGlobal: '全局',
    scopeProject: '项目',
    searchPlaceholder: '按名称过滤',
    btnNew: '+ 新建技能',
    btnAiOpen: '打开 AI',
    btnAiClose: '关闭 AI',

    applyBar: {
      target: '应用目标工具:',
      checkUpdates: '检测更新',
      checking: '检测中…',
      updatesAvailable: '{updates} / {total} 可更新',
      allUpToDate: '{total} 个技能已是最新',
      appliedOk: '已把 {name}@{version} 落到 {tool}',
      appliedPartial: '部分失败: {detail}',
      errDefault: '应用失败',
    },

    editor: {
      titleNew: '新建技能',
      titleEdit: '编辑技能',
      name: '名称',
      nameHint: '英文短名,如 review-pr',
      version: '版本',
      versionHint: '0.1.0',
      scope: '作用域',
      projectId: '项目 ID',
      description: '描述',
      descriptionHint: '≥ 10 字符',
      // 2026-06-27 新增:详情页 description 前的小 label(与 triggers-label 风格一致)
      descShort: '描述',
      triggers: '触发词',
      triggersHint: '用逗号分隔',
      triggersHintPlaceholder: 'review pr, code review',
      // 2026-07-12 增:触发词改为可选,label 注明"可选"、列表为空时给占位提示
      triggersOptional: '可选',
      triggersEmptyHint: '未填写触发词,skill 不会按关键词自动触发',
      body: '正文 (Markdown,frontmatter 会自动拼)',
      // 2026-06-26 新增:作用域区改造 + 适用工具
      scopeGlobal: '全局',
      scopeProject: '项目',
      scopeGlobalHint: '对所有项目可见',
      projectPick: '请选择项目',
      applyTools: '适用工具',
      applyToolsHint: '勾选后会自动启用',
      applyToolsNone: '暂未勾选,创建后只在 skillbox 库中',
      applyToolsSelected: '已选 {n} 个',
      errProjectRequired: '请先选择项目',
      applyAllSuccess: '已在 {n} 个工具上自动启用',
      applyPartialFailed: '{ok}/{total} 个工具启用成功,失败: {fails}',
      // 2026-06-26 新增:编辑保存后同步拷贝到已启用命中的提示
      syncAllSuccess: '已同步到 {n} 个生效位置',
      syncPartialFailed: '{ok}/{total} 个生效位置同步成功',
      syncNone: '「{name}」已保存,但还没在任何工具/项目上启用,需要时去作用域区启用',
      errNameEmpty: '名称不能为空',
      errVersionEmpty: '版本不能为空',
      errDescriptionEmpty: '描述不能为空',
      errDescShort: '描述至少 10 个字符',
      errTriggersEmpty: '触发词至少填一个',
      // 2026-07-13 增:frontmatter 表单 + InlinePanel 编辑保存相关
      notReady: '编辑器尚未就绪,请稍候再试',
      saveOk: '已保存 {name}',
      created: '已创建 {name}@{version}',
      createdRefreshFailed: '新建完成但刷新失败:{msg}',
      frontmatterDialogTitle: '编辑 / 新建 frontmatter',
      triggerPlaceholder: '触发词 #{idx}',
      deleteTrigger: '删除第 {idx} 个',
      addTrigger: '添加触发词',
      descriptionRequiredPlaceholder: '技能说明(必填)',
      licensePlaceholder: '(可选,例如 MIT)',
      triggersList: '触发词(列表)',
      saving: '保存中...',
    },

    // 2026-07-03 增:apply / batch 响应的统一判定 toast 文案(全文件复用)。
    // 之前 skills.editor.applyPartialFailed 是 editor 内的兜底,新的 skills.apply
    // 命名空间用于跨页面(首页 doApplyOne、编辑保存同步、创建时勾选工具)的
    // 统一"部分失败"展示,带 {detail} 多行失败明细。
    apply: {
      allOk: '已成功应用到 {n} 个工具',
      partialFailed: '{ok}/{total} 个工具启用成功,失败明细:\n{detail}',
    },

    applyHistory: {
      title: '最近应用历史',
      count: '{count} 条',
      undone: '撤销',
      undoing: '撤销中…',
      applied: '已应用',
      rolledBack: '已回滚',
      failed: '失败',
    },

    // 2026-07-17 删:tag 块(对应旧 ctag 弹窗,改走 go-git VersionHistoryModal)。

    test: {
      title: '最近测试结果',
      errPrefix: '测试失败:',
      passed: '通过',
      failed: '失败',
      errored: '出错',
      skipped: '跳过',
      confirmRun: '对技能 "{name}@{version}" 跑一次测试?(静态 + 脚本 + AI)',
    },

    list: {
      title: '技能',
      colName: '名称',
      colVersion: '版本',
      colSource: '来源',
      colProject: '项目',
      colUpdated: '更新时间',
      colActions: '操作',
      btnApply: '应用',
      applying: '应用中…',
      btnTest: '测试',
      testing: '测试中…',
      btnEdit: '编辑',
      btnTag: '打标签',
      btnDelete: '删除',
      confirmDelete: '确定删除技能 "{name}@{version}" ?',
      emptyTitle: '该作用域下还没有技能',
      emptyHint: '点右上角"+ 新建技能"开始,或去首次配置从已装工具导入',

      // 左右布局新增
      btnNewSkill: '新建',
      btnNewSkillTitle: '新建技能',
      btnImportSkill: '导入',
      btnImportSkillTitle: '从已装工具导入',
      searchTitle: '按名称过滤',
      selectToView: '从左侧选一个技能查看详情',
      // 2026-07-08 增:空状态引导用户新建第一个技能
      noSkillTitle: '还没有任何技能',
      noSkillHint: '点上方"新建"创建第一个技能,或"导入"从已装工具拉取',
      noSkillBtnCreate: '新建技能',
      noSkillBtnImport: '从已装工具导入',
      noFilesHint: '该技能没有可渲染的正文',
      scopeLabel: '作用域',
      scopeGlobalChip: '全局',
      scopeProjectChip: '项目',
      scopeToolsRow: '工具',
      scopeTargetsRow: '生效位置',
      scopeEmpty: '该技能尚未写入任何工具/位置',
      scopeHitCount: '{n} 处生效',
      scopeSelectToolFirst: '先在「工具」行点选一个工具,再操作生效位置',
      scopeForTool: '对 {tool} 生效',
      scopeToolSelected: '已选 {tool}',
      applyConfirmTitle: '启用作用域',
      applyConfirmMessage: '将 skill "{name}" 复制到 {tool} · {scope}?',
      applySuccess: '已启用:{path}',
      applyFailed: '启用失败:{msg}',
      unapplyConfirmTitle: '停用作用域',
      unapplyConfirmMessage: '将从 {tool} · {scope} 删除 skill "{name}"(走 apply/undo 还原 PreSnapshot),继续?',
      unapplySuccess: '已停用:{path}',
      unapplyFailed: '停用失败:{msg}',
      appliedGlobal: '{tool} 已全局应用',
      applying: '启用中…',
      unapplying: '停用中…',
      tagsEmpty: '还没有标签,点右上角打一个',
      bodyEmpty: 'SKILL.md 还没有正文',
      bodyTitle: '正文',
      bodyEditing: '编辑正文 (Markdown)',
      tooltipTest: '测试',
      tooltipTag: '打标签',
      tooltipOpenFolder: '在文件夹打开',
      tooltipDelete: '删除',
      copied: '已复制',
      openFailed: '打开失败: {msg}',
      goOnboarding: '去导入',

      // 2026-06-29 增:分组树形 UI + 右键菜单 + 拖拽
      treeEmpty: '还没有技能,点上方"新建"开始',
      treeRootHint: '右键空白处新建分组,或拖拽 skill 到此处',
      ctxNewGroup: '新建分组',
      ctxNewSubgroup: '新建子分组',
      ctxDeleteGroup: '删除分组',
      ctxRename: '重命名',
      ctxOpenFolder: '在文件夹中打开',
      ctxCopyPath: '复制路径',
      // 2026-07-17 改:tag 弹窗下线,改成"版本历史"弹窗(go-git commit 时间线)。
      ctxVersion: '查看版本历史',
      ctxDelete: '删除',
      ctxMoveTo: '移动到…',
      // 2026-07-03 改:首页分组只支持单级,不再支持多级。groupNamePrompt
      // 提示从"可多级,用 / 分隔"改为"只支持单级,不能含 /"。
      groupNamePrompt: '请输入分组名(只支持单级,不能含 /)',
      groupNamePromptSub: '请输入分组名(只支持单级,不能含 /)',
      groupInvalid: '分组名非法(不能含 .. 或 / 开头/结尾)',
      groupCreateFailed: '新建分组失败: {msg}',
      groupDeleteConfirm: '确定删除分组「{name}」吗?',
      groupDeleteConfirmCascade: '该分组下还有 {n} 个 skill,会一并删除。是否同步清理这些 skill 在 5 个工具目录中的副本?',
      groupDeleteCascadeHint: '同步删除这些 skill 在工具目录(5 个工具 × 全局 / 各项目)中的副本',
      groupDeleteFailed: '删除分组失败: {msg}',
      // 2026-06-29 增:重命名分组
      groupRenamePrompt: '重命名分组「{name}」',
      groupRenameHint: '只改这一段,父路径不变;合法字符:小写字母 / 数字 / - / _',
      groupRenameOk: '已重命名为「{name}」',
      groupRenameFailed: '重命名失败: {msg}',
      groupRenameConflict: '同层已存在同名分组',
      groupRenameNotFound: '原分组不存在',
      skillDeleteConfirm: '确定删除技能「{name}」吗?',
      skillDeleteCascadeHint: '同步删除该 skill 在工具目录(5 个工具 × 全局 / 各项目)中的副本',
      skillDeleteFailed: '删除失败: {msg}',
      skillCascadeOk: '已删除「{name}」及 {n} 处工具目录副本',
      skillCascadePartial: '已删除「{name}」,{n} 处工具目录清理失败:{detail}',
      skillCascadeSkipped: '已删除「{name}」(工具目录副本未清理)',
      skillTagOpenFailed: '打标签失败: 请先选中一个 skill',
      skillOpenFolderOk: '已在文件夹中打开',
      skillOpenFolderFailed: '打开失败: {msg}',
      moveFailed: '移动失败: {msg}',
      moveTargetExists: '目标位置已存在同名 skill',
      moveSameGroup: '源和目标分组相同,无需移动',
      // 2026-06-29 增:把分组拖到自己的子分组下(物理上无法执行,会死循环)
      moveIntoDescendant: '不能把分组挪到自己的子分组下',
      // 2026-06-29 增:根目录拖入视觉提示 + no-op 提示
      dropToRoot: '放到根',
      alreadyAtRoot: '目标已在根,无需移动',
      // 2026-07-08 增:skill 不能拖到另一个 skill 上,只能拖到分组或根
      dropOnSkillNotAllowed: 'skill 只能拖到分组或根目录下',
      loadingTree: '加载技能树…',
    },

    // 2026-07-04 增:首页技能文件浏览器(在 skills 顶层,不在 list 内)
    fileBrowser: {
      open: '浏览文件',
      files: '{n} 个文件',
      readOnly: '只读',
      readOnlyHint: 'SKILL.md 在抽屉中只读,请在主区编辑',
      unsaved: '有未保存的修改,确定关闭吗?',
      unsavedShort: '未保存',
      noFile: '未选中文件',
      pickOne: '从左侧选择一个文件查看',
      saved: '已保存 {path}',
      saveFailed: '保存失败: {msg}',
      discard: '放弃',
      save: '保存',
      saving: '保存中',
      binaryTitle: '不支持在线预览',
      binaryHint: '二进制文件(.{ext})可在文件夹中查看',
      largeTitle: '文件过大,不支持在线预览',
      largeHint: '文件大小 {kb} KB,可在文件夹中查看',
      openInFolder: '在文件夹打开',
      // 2026-07-05 增:磁盘文件损坏提示(后端 corrupted_file 错误时弹)
      corruptedHint: '技能「{name}」的 SKILL.md 文件已损坏({hint})。请检查磁盘文件内容,或从备份恢复。',
      // 2026-07-13 增:补齐 SkillFileInlinePanel + SkillScopePanel 用到的所有文案
      skillDirectory: 'skill 目录',
      files: '{n} 个文件',
      pickOneToBrowse: '请选择一个文件开始浏览',
      renderError: '技能详情加载出错',
      showOutline: '显示大纲',
      hideOutline: '隐藏大纲',
      outline: '大纲',
      viewFrontmatter: '查看 frontmatter',
      ariaLabel: '文件浏览器',
      ctxNewFile: '新建文件',
      ctxNewFolder: '新建文件夹',
      ctxRenameFile: '重命名文件',
      ctxRenameFolder: '重命名文件夹',
      ctxDeleteFile: '删除文件',
      ctxDeleteFolder: '删除文件夹',
      openInExplorer: '在文件浏览器中打开',
      newFileTitle: '新建文件(将在指定目录下创建)',
      newFolderTitle: '新建文件夹(将在指定目录下创建)',
      renameFileTitle: '重命名文件',
      renameFolderTitle: '重命名文件夹',
      deleteFileTitle: '删除文件',
      deleteFileConfirm: '确定要删除「{name}」吗?此操作无法撤销。',
      deleteFolderConfirm: '确定要删除目录「{name}」吗?',
      deleteFolderChildrenWarning: '该目录下还有 {n} 个文件会被一并删除,此操作无法撤销。',
      newFilePlaceholder: '文件名(如 notes.md)',
      newFolderPlaceholder: '目录名(如 examples)',
      modifiedTitle: '文件已修改',
      discardPrompt: '文件已被修改,切换前请选择如何处理:',
      discardSaveHint: '保存修改:写盘后再切换',
      discardDropHint: '放弃修改:丢弃本地编辑,加载目标 skill / 文件',
      discardCancelHint: '取消:留在当前页面继续编辑',
      discardChanges: '放弃修改',
      saveChanges: '保存修改',
      modifiedTitleDirty: '文件已被修改',
      incompleteFilesWarning: '提示:文件列表为空,只提交了当前文件,保存后其他文件会丢失 — 请等待目录加载完成后再保存。',
      noSkillSelected: '当前未选中技能',
      noFrontmatter: '无 frontmatter',
      modifiedState: '● 未保存',
      modifiedShort: '● 未保存',
      // 文件名 / 目录名校验
      validation: {
        nameRequired: '名称不能为空',
        invalidSeparator: '名称不能含 / 或 \\',
        invalidDotName: '名称不能为 . 或 ..',
        invalidSKILL: 'SKILL.md 由系统管理,不能直接新建',
        duplicateName: '已存在同名文件/目录',
        duplicateFile: '同目录下已存在同名文件',
      },
      createdFile: '已新建文件「{name}」',
      createdDir: '已新建目录「{name}」',
      renamed: '已重命名为「{name}」',
      deletedItem: '已删除「{name}」',
      sourceDirMissing: '无法定位到磁盘目录,缺少 source_dir',
      skillDirFolder: '技能目录',
      // CsvViewer
      csvPreviewLimit: '文件共 {total} 行,仅预览前 {preview} 行;编辑模式看完整内容',
      csvEmpty: '空文件或无可解析行',
      // OfficeViewer
      officeParseFailed: '文档解析失败',
      officeParseHint: '文件可能已损坏,或二进制内容还原不完整。请用在文件夹中打开在原生应用查看。',
      // FileTreeNode
      emptyFolder: '空文件夹',
      emptyShort: '空',
      largeShort: '大',
      unsavedChanges: '有未保存的修改',
    },

    ai: {
      header: 'AI 助手',
      clear: '清空',
      empty: '先选一个预设(优化 frontmatter / 检验描述 / 润色正文 / 查重复 / 安全检查),再发问。',
      hintNoProvider: '暂未配置 AI 提供方或内置预设',
      pickFirst: '请先在上方选一个预设。',
      pickedDedupe: '请在输入框里把要对比的若干技能全文贴进来(每个用 \\n\\n---\\n\\n 分隔),我会给出重叠度评分。',
      pickedPreset: '已选择预设:「{title}」。{description}\n把上下文(可空)和额外要求贴到下方,点发送即可。',
      roleUser: '你',
      roleAssistant: 'AI',
      copy: '复制',
      inputPlaceholderHint: '补充说明(可空)',
      inputPlaceholderNoPreset: '先选预设',
      send: '发送',
      stop: '停止',
      noExtraInput: '(无额外输入,只基于上下文)',
      errorTag: '[错误] {msg}',

      // 2026-07-12 增:AI 操作弹窗(全局,与下方 aiDialog.* 共享)。AIPanel 仍可能在用,
      // aiDialog.* 是新弹窗专用 key。
    },

    // 2026-07-13 增:AI 右侧对话面板(替换原大纲区域,接替旧 AIDialog 弹窗入口)。
    // 文件工具栏点 AI 图标 → 右侧面板从大纲切换为聊天面板;标签栏提供翻译/检测快捷入口。
    aiPanel: {
      open: '打开 Agent Dog',
      close: '关闭 Agent Dog',
      tagTranslate: '翻译',
      tagReview: '检测',
      inputPlaceholder: '向 Agent Dog 提问…(Shift+Enter 换行)',
      send: '发送',
      stop: '停止',
      closeBtn: '关闭',
      switchToOutline: '切换到大纲',
      clearHistory: '清空对话',
      // 2026-07-14 增:历史对话相关
      history: '历史对话',
      // 2026-07-14 v2 改:清空改名"新建对话"(归档当前会话到历史)
      newConv: '新建对话',
      archiveFailed: '保存到历史失败',
      deleteConv: '删除对话',
      deleteFailed: '删除对话失败',
      loadFailed: '加载失败',
      historyDialog: {
        title: '历史对话',
        loading: '加载中…',
        empty: '暂无历史对话',
      },
      roleYou: '我',
      roleAI: 'Agent Dog',
      emptyHint: '我是 agent dog,点击下方标签快速调用噢',
      noProvider: '尚未配置 AI 模型,请先到「设置 → AI 模型」里配一个。',
      noContent: '当前文件无内容,无法调用 Agent Dog。',
      apply: '应用',
      reject: '拒绝',
      applied: '已应用',
      rejected: '已拒绝',
      applyFailed: '应用失败:{msg}',
      reviewTitle: '检测结果',
      noIssues: '未发现明显问题',
      translateDialog: {
        title: '选择目标语言',
        desc: '确定后,系统会把翻译提示词 + 当前文档原文填到下方输入框,你点击发送即可。',
        confirm: '确定',
        cancel: '取消',
      },
      // 2026-07-17 改:translatePromptTemplate / reviewPromptTemplate /
      //   customPromptHint 三段含 "needs_apply": 字面 JSON 的字符串会触发
      //   vue-i18n ICU 解析器抛 "Invalid token in placeholder" 错误,导致
      //   SkillFileInlinePanel.onErrorCaptured 误显示 "技能详情加载出错"。
      //   模板已挪到 frontend/src/core/ai/promptTemplates.js。
      // 2026-07-13 增 v2:JSON 解析失败兜底 + 全屏编辑相关文案
      retrying: '正在重新生成…({left} 次剩余)',
      parseFailed: 'AI 返回格式异常,已重试 3 次仍未成功,仅作展示无法应用',
      truncated: 'AI 输出过长,已截断',
      fullscreenEdit: '全屏编辑',
      fullscreenEditTitle: '全屏编辑输入内容',
      fullscreenSave: '保存并返回',
    },

    // 2026-07-13 增:SkillScopePanel 作用域区文案(从组件内 LABEL_* 常量迁移过来)。
    scope: {
      title: 'skill 作用域',
      global: '全局',
      // 2026-07-16 删:projectPrefix(原来用于拼接"项目 #N")。
      // SkillScopePanel 现在直接从 projects 数据表读取用户在项目页配置的 name,
      // 不再硬拼 i18n 文案,避免显示成语言标识。保留空行注释避免结构变动。
      empty: '该技能尚未写入任何工具/位置',
      loading: '加载中...',
      enable: '启用作用域',
      disable: '停用作用域',
      loadError: '作用域加载出错',
      retry: '重试',
      enabled: '已启用',
      disabled: '已停用',
      partialFailed: '部分失败',
      enableFailed: '启用失败',
      disableFailed: '停用失败',
      enableSuccess: '已启用 {tool} · {scope}',
      disableSuccess: '已停用 {tool} · {scope}',
      disableConfirm: '确定要从 {tool} · {scope} 删除 skill「{name}」?',
      enableConfirm: '确定要把 skill「{name}」复制到 {tool} · {scope}?',
      toolCount: '{n} 个工具',
      toolCountShort: '{n} 个工具',
      globalEnabledTip: '已启用全局作用域',
      globalAgent: '全局 Agent Skill',
      globalAgentTip: '同步到 ~/.agents/skills/ 共享池(所有工具均可读取)',
      globalAgentInfoTip: '查看全局 Agent 说明',
      globalAgentFolderTip: '在文件浏览器中打开 ~/.agents/skills/',
      globalAgentInfoTitle: '全局 Agent Skill',
      globalAgentInfoDesc: '把 skill 写入 ~/.agents/skills/ 后,所有声明该目录作为个人级 skills 池的 AI 工具都能自动读取(无需复制到每个工具的目录)。',
      globalAgentCompatibleToolsTitle: '适配 ~/.agents/skills/ 的工具(2026-07-12)',
      globalAgentSupported: '已支持',
      globalAgentPartial: '部分支持',
      globalAgentEmpty: '暂未发现适配工具。',
      globalAgentEnabled: '已同步到 ~/.agents/skills/{name}/',
      globalAgentDisabled: '已从 ~/.agents/skills/{name}/ 移除',
      globalAgentToggleFailed: '切换全局 Agent 失败:{msg}',
      globalAgentPathFailed: '无法获取 ~/.agents/skills/ 路径',
      openFolderFailed: '打开文件夹失败:{msg}',
      globalAgentDirToast: '全局 Agent 目录: {url}',
      // 各工具支持备注(给 info 弹窗用)
      toolNotes: {
        vscode: '文档明确支持个人级',
        antigravity: '官方支持',
        claude: '项目级支持,个人级实际走其他目录',
        codex: '命令行沿用开放标准',
        qwen: '官方支持技能标准',
        cursor: '主要走其他目录,个人级路径尚未官方文档化',
        opencode: '官方文档未明确支持个人级路径',
        other: '官方文档未提及该路径',
      },
      dirToast: '全局 Agent 目录: {path}',
    },

    // 2026-07-12 增:AI 弹窗(全局独立弹窗,替代旧 AIPanel 嵌入)。
    aiDialog: {
      title: 'AI 操作',
      subtitle: '基于大模型的常用工具',
      btnOpen: 'AI',
      actionsTitle: '请选择一个操作',
      actions: {
        translate: '翻译 Skill',
        translateDesc: '把当前技能全文翻译为目标语言',
        optimize: '优化 Frontmatter',
        optimizeDesc: '改写 name / description / triggers',
        comingSoon: '敬请期待',
      },
      // translate 子面板
      translate: {
        title: '翻译 Skill',
        desc: '调用 AI 把当前技能的 SKILL.md 翻译到目标语言,保留 frontmatter 字段名与代码块。',
        targetLang: '目标语言',
        promptLabel: '原始提示词',
        promptHint: '原始提示词模板;切换目标语言时,里面的 {target_lang} 占位符会自动替换。',
        promptCopy: '复制提示词',
        promptCopied: '已复制',
        // 已自定义提示词徽标(用户改过)
        promptCustomized: '已自定义',
        // 把自定义内容改回默认模板
        promptReset: '重置为默认',
        // 2026-07-13 增:翻译面板相关提示
        promptEmpty: '提示词为空,请填写或点重置为默认',
        startInstruction: '请按系统提示里的规则开始翻译',
        // 内置的"原始提示词"模板(含 {target_lang} 占位符,弹窗实时替换 + 显示)。
        // 这段是「给用户看的预览」 — 真正发给 LLM 的 system prompt 由后端 preset 注入,
        // 这里展示出来只是让用户对翻译质量心里有数(可点击「复制提示词」拿去别处 debug)。
        promptTemplate:
`请把下面的「Claude / Codex Skill」SKILL.md 文档完整翻译成目标语言:{target_lang}。

# 翻译规则(必须严格遵守)
1. frontmatter 字段名(name / version / description / triggers)保留英文原文,不翻译字段名;
2. frontmatter 的 description 字段值如果是一段中文描述,翻译成目标语言;
3. Markdown 中以三个反引号 \`\`\` ... \`\`\` 包起来的代码块、命令、输出例子,**完全不翻译**,原样保留;
4. Markdown 标题层级、列表、链接、图片、表格结构保留不动,只翻译里面的文字;
5. 翻译风格:专业、简洁、保持技术写作的语气;不要加注释、不要解释、不要寒暄;
6. 输出格式:只输出翻译后的完整 SKILL.md 文档,不要任何开场白、不要解释、不要附加段落;
7. 如果 SKILL.md 是英文写的,description 字段值就是英文的,这种情况也照常翻译 description 字段值。

# 待翻译的 SKILL.md
\`\`\`markdown
{skill_md}
\`\`\`

# 输出(只输出翻译后的 markdown 正文)
`,
        submit: '开始翻译',
        submitting: '翻译中…',
        stop: '停止',
        resultTitle: '翻译结果',
        copyResult: '复制结果',
        applyToEditor: '应用',
        applied: '已应用',
        applyFailed: '应用失败:{msg}',
        noContext: '(当前没有选中技能,无法获取 SKILL.md)',
      },
      langs: {
        'zh-CN': '简体中文',
        'zh-TW': '繁體中文',
        'en-US': 'English',
        'ja-JP': '日本語',
        'ko-KR': '한국어',
        'fr-FR': 'Français',
        'de-DE': 'Deutsch',
        'es-ES': 'Español',
      },
      providerMissing: '尚未配置 AI 模型,请先到「设置 → AI 模型」里配一个。',
      providerMissingTitle: '需要先配置 AI 模型',
    },

    // 2026-07-13 增:AI 应用 / 翻译相关(toast / 错误提示)
    aiApply: {
      noSkill: '应用失败:当前未选中技能',
      empty: '应用失败:AI 输出为空',
      applied: '已应用(已写回 {scope}/{name})',
      failed: '应用失败:{msg}',
    },

    // 2026-07-13 增:RichTextEditor(详情区正文的所见即所得编辑器)文案
    richText: {
      placeholder: '开始输入内容',
      heading1: '标题',
      heading2: '标题',
      heading3: '标题',
      bold: '加粗',
      italic: '斜体',
      strike: '删除线',
      inlineCode: '行内代码',
      bulletList: '无序列表',
      orderedList: '有序列表',
      blockquote: '引用',
      codeBlock: '代码块',
      link: '链接',
      unlink: '取消链接',
      insertImageTip: '插入图片,填地址',
      imageUrl: '图片地址',
      imageAlt: '替代文本,可选',
      insert: '插入',
      undo: '撤销',
      redo: '重做',
    },

    // 2026-07-13 增:HelloWorld 调试页(根路由)文案
    helloWorld: {
      webNoLocalPort: 'Web 模式,无本地端口',
      dualDeployment: '双部署:桌面端 + Web',
      deploymentMode: '部署形态',
      appName: '应用名',
      localBackendPort: '本地后端端口',
      sameOrigin: '空,同源',
      healthCheck: '健康检查',
      pingBackend: 'ping 后端',
      enableDebugLog: '开日志',
      storeStatus: '应用状态',
      httpScaffoldHint: '接口样板,业务继续按写,路由自动注册',
    },

    // 2026-07-13 增:AISettingsPanel(AI 模型设置)供应商类型文案
    aiProvider: {
      official: '官方',
      openAI: '官方',
      anthropic: '官方',
      openAICompat: '兼容、硅基等',
    },

    // 2026-07-16 改:卡片上的全局 Agent 标签按用户反馈移除(全局 Agent skill
    // 卡片前的 mdi:earth 图标已经能代表"全局 Agent"语义,再叠 badge 会挤压
    // skill name 横向空间)。globalAgentTip 保留 — 用作 TreeNode 图标的 hover
    // title 提示。
    treeNode: {
      globalAgentTip: '该技能位于全局 agents 目录,所有工具可自动读取',
      dropHere: '放到此处',
    },
  },

  projects: {
    title: '项目',
    subtitle: '登记项目根目录,后续技能可绑定到项目作用域,走项目级覆盖。',
    btnImport: '导入项目',
    btnImportTitle: '从本地文件夹导入一个项目',
    btnEdit: '编辑',
    btnEditTitle: '编辑项目信息',
    editFormTitle: '编辑项目',
    btnPickAgain: '重新选择目录',
    btnCancel: '取消',
    searchPlaceholder: '按名称过滤',
    formTitle: '导入项目',
    formHint: '已根据所选目录自动解析名称与别名,可按需修改。',
    inspecting: '解析中…',
    name: '名称',
    nameHint: '显示名,如 My App',
    alias: '别名',
    aliasHint: '唯一别名,英文短码',
    rootPath: '根路径',
    rootPathHint: '项目根绝对路径',
    description: '描述',
    descriptionHint: '可选,描述项目用途',
    errRequired: '名称 / 别名 / 根路径都不能为空',
    listTitle: '项目列表',
    colId: 'ID',
    colName: '名称',
    colAlias: '别名',
    colRootPath: '根路径',
    colDescription: '描述',
    colActions: '操作',
    confirmDelete: '确定删除项目 #{id} ?',
    empty: '还没有登记项目。',
    emptyHint: '点右上角"导入项目",从本地选一个目录开始',
    noTools: '该项目尚未应用任何工具',
    openInFinder: '在 Finder 中打开',
    openFailed: '打开文件夹失败:{msg}',
    scannedAt: '扫描于 {time}',
    scanFailed: '扫描失败',
    toolSkillsTitle: '{project} · {tool} 的 skills ({count})',
    skillPath: '路径',
    // 工具 skill 弹窗里的"跳转"和"删除"两个操作图标
    skillActionReveal: '在文件管理器中打开该 skill 目录',
    skillActionDelete: '删除该 skill 目录(物理删除)',
    skillDeleteConfirm: '确定删除 skill "{name}"?此操作会永久删除磁盘目录:\n{path}',
    skillDeleteFailed: '删除失败:{msg}',
    skillRevealFailed: '跳转失败:{msg}',
  },

  // 2026-07-09 改:MarketView 改回「卡片 + 在浏览器中打开」方案。
  // iframe 嵌入被 skillhub 站点 CORS 拒(同源策略,代理解决不了),放弃。
  // 保留:title / subtitle / btnOpenInBrowser / cards.* 两张站点卡的描述
  //
  // 2026-07-10 改:
  //   - skillhub 国内源更名为 "skillhub-cn",i18n 同步换成新名字
  //   - tab name 改名(SkillHub → SkillHub-CN),
  //     同时按 getter 函数(后续前端 MarketView 可以按 locale 自动选 tab)
  //   - GitHub tab 增 「知名 skill 快捷浏览」列表 + 「粘贴」按钮 文案
  market: {
    title: '三方市场',
    subtitle: '浏览 skillhub.cn / skills.sh / GitHub 等三方源,粘详情页 URL 一键装到 skill-box。',
    btnOpenInBrowser: '在浏览器中打开',
    btnOpenInBrowserTip: '在系统浏览器打开 {name} 站点',
    // 2026-07-10 增:粘贴并安装按钮(主按钮「安装」右侧)
    // 2026-07-10 改:改名「粘贴」→「粘贴并安装」,粘贴成功自动触发安装流程
    btnPaste: '粘贴并安装',
    btnPasteTitle: '读取系统剪贴板文本,粘贴到上方输入框并直接开始安装',
    btnPasteEmpty: '剪贴板为空,无法粘贴,请先复制一个 skill 详情页 URL',
    btnPasteFailed: '读取剪贴板失败:{msg}',
    // 2026-07-10 改:v5.1 去掉 btnPasteSuccess(粘贴成功已经在装,无需 toast 提示)
    // 站点卡片描述(在卡片里展示)
    cards: {
      skillhubDesc: '中国用户的 Skills 社区,按 curated_score 排序的精选技能集合。',
      skillsshDesc: 'Vercel 托管的 AI 技能排行榜,按热度(1H / change)实时排序。',
      githubDesc: 'GitHub 上的 agent skill 仓库,直接走 raw 内容 URL 装到本地。',
    },
    // 2026-07-10 增:GitHub tab 知名 skill 仓库快捷浏览块
    githubFamous: {
      title: '知名 skill 仓库',
      desc: '下面是几个社区知名的、开源在 GitHub 上的 agent skill 仓库,点按钮在浏览器里浏览。',
      btnOpen: '打开',
    },
    // 2026-07-09 增:输入框一键安装
    guide: {
      title: '如何安装到 skill-box',
      skillhub: {
        desc: '在 skillhub.cn 浏览找到想要的 skill → 复制浏览器地址栏 URL → 粘贴到下方输入框。',
        examples: '输入示例',
        examplesList: [
          'https://skillhub.cn/skills/code-review',
          'https://skillhub.cn/skills/commit-msg',
        ],
      },
      skillssh: {
        desc: '在 skills.sh 浏览找到 skill → 复制详情页 URL(或 GitHub blob URL)→ 粘贴到下方输入框。',
        examples: '输入示例',
        examplesList: [
          'https://skills.sh/anthropics/skills/pdf',
          'https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md',
        ],
      },
      github: {
        desc: '在 GitHub 浏览 agent skill 仓库 → 找到 SKILL.md → 复制 blob URL → 粘贴到下方输入框。',
        examples: '输入示例',
        examplesList: [
          'https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md',
          'https://github.com/anthropics/skills/blob/main/skills/code-review/SKILL.md',
        ],
      },
    },
    input: {
      label: '粘贴详情页 URL',
      placeholderSkillhub: 'https://skillhub.cn/skills/{slug}',
      placeholderSkillssh: 'https://skills.sh/{owner}/{repo}/{skill} 或 GitHub blob URL',
      placeholderGithub: 'https://github.com/{owner}/{repo}/blob/{branch}/{path}/SKILL.md',
      // 2026-07-10 改:「装到 skill-box」→「安装」(用户要求简化文案,
      // 「skill-box」已经写在页面标题里,按钮文字不用重复)
      btnInstall: '安装',
      btnInstalling: '安装中…',
      errInvalidInput: '输入格式无法识别:每个 tab 只接受对应市场的详情页 URL',
      errSource: '找不到对应的市场源',
      // 2026-07-10 增:404 / 上游资源无效
      // (用户报告 topnews / ima-skills 报「下载失败」误导,根因是 SKILL.md 缺 frontmatter)
      // 文案覆盖 4 种上游语义:slug 真不存在 / slug 下架 / zip 缺 SKILL.md / zip 不合法
      errSkillMalformed: '该 skill 文件格式有问题({msg})。可能:作者发布时漏了 frontmatter、上传 SKILL.md 内容为空、zip 包损坏。该 skill 暂时装不上',
      // 2026-07-10 改:404 细分 — errSkillNotFound(slug 不存在 / 已下架) / errSkillMalformed(走 422,文案见上)
      errSkillNotFound: '该 skill 不存在({msg})。可能:URL 里 slug 拼错、上游已下架,请确认 URL 后重试',
      errPull: '下载失败:{msg}',
      // 2026-07-09 增:前端 timeout 单独提示
      errTimeout: '请求超时(后端下载慢):{msg}。可以重试,或去浏览器手动下好后从「本地导入」装入。',
      errGeneric: '安装失败:{msg}',
      // 2026-07-13 增:补充
      clickToFill: '点击填入',
      rateLimitHint: '疑似限流。等几分钟再点重试,或去浏览器手动下载后从首页本地导入',
    },
    progress: {
      resolve: '解析输入…',
      download: '下载 skill…',
      extract: '解压并校验…',
      write: '写入 skill-box store…',
      done: '安装完成',
      // 2026-07-09 增:'fail' 阶段(报错时保留进度条,告诉用户卡哪步)
      fail: '安装失败',
      // 2026-07-09 增:子步骤文字(进度条下方展示当前在干什么)
      hintResolve: '正在解析 URL,匹配源类型…',
      hintDownload: '正在下载 zip 包(从 GitHub codeload)…',
      hintExtract: '正在解压并校验 SKILL.md…',
      hintWrite: '正在写入本地 skill-box store…',
      hintDone: '已装入,准备跳转',
      // fail 阶段 hint(显示具体卡哪步 + 错误码,便于排查)
      hintFailResolve: '解析阶段失败,未真正发请求',
      hintFailDownload: '下载阶段失败(可能 zipball 404 / 网络 / 限流)',
      hintFailExtract: '解压阶段失败(SKILL.md 解析异常)',
      hintFailWrite: '写入 store 阶段失败(磁盘 / 权限)',
      hintFailUnknown: '未知阶段失败,看下方错误条',
    },
    success: {
      msg: '✅ 已装入 {name} v{version}',
      goHome: '去首页查看',
    },
    // 2026-07-09 增:同名 skill 冲突确认
    conflict: {
      title: 'skill 已存在',
      desc: '本地已有同名 skill「{name}」(v{existingVersion}@{existingPath}),要怎么处理?',
      overwrite: '覆盖',
      overwriteTip: '删除旧版本,装新版本(同名文件会被替换)',
      rename: '另存为',
      renameTip: '自动加 -2 / -3 后缀,跟旧版本并存',
      cancel: '取消',
    },
  },

  onboarding: {
    title: '导入技能',
    subtitle: '扫描本机 5 个 AI 编程工具的技能目录,把发现的技能勾选导入到 Skill Box 自己的 store(全局作用域)。',
    btnRescan: '重新扫描',
    btnRescanning: '扫描中…',
    btnRescanTitle: '重新扫描 5 个 adapter',
    // 2026-07-01 增:弹窗顶部 tab 切换(扫工具 vs 从本地导入)
    // 2026-07-18 改:tab 顺序调整 + 「扫描工具」改名「工具」+ 新增「从三方导入」首位
    tabs: {
      market: '从三方导入',
      scan: '工具',
      global: '全局目录',
      local: '从本地导入',
    },
    // 2026-07-18 增:三方市场导入面板 i18n(参考 onboarding.local.* 结构)
    // sources 列表的简介 + 输入框 placeholder + 错误文案
    market: {
      descSkillhub: 'SkillHub 中文社区市场,国内访问快',
      descSkillssh: 'skills.sh 国际市场,聚焦 skills 详情目录',
      descGithub: 'GitHub 主流仓库的 SKILL.md 详情页',
      inputPlaceholder: '粘贴详情页 URL(skillhub.cn / skills.sh / github.com)',
      btnImport: '导入',
      btnImporting: '导入中…',
      clear: '清空',
      errEmpty: '请粘贴一个详情页 URL',
      tip: '复制上面任何一个源的详情页链接,粘贴到输入框即可一键导入',
      // 2026-07-18 增:输入示例区(三个来源各一条)
      examplesLabel: '输入示例(点击直接填入)',
      detectedSource: '已识别为 {name}',
      gotoSite: '前往 {name} 官网',
      fillExample: '填入示例',
      // 2026-07-18 增:GitHub 仓库子卡下方简介(谁提供的)
      repoAnthropics: 'Anthropic 官方维护,Claude 生态技能',
      repoVercelLabs: 'Vercel Labs 维护,前端 / Next.js 相关技能',
      repoGoogle: 'Google 官方仓库,搜索 / 工程类技能',
      // 粘贴并导入
      btnPasteAndImport: '粘贴并导入',
      btnPasteTitle: '读取剪贴板并自动安装',
      btnPasteFailed: '读取剪贴板失败:{msg}',
      btnPasteEmpty: '剪贴板为空,请先复制一个详情页 URL',
      // 2026-07-18 增:导入前本地同名检测
      localCheck: {
        exists: '本地已存在同名 skill「{name}」',
        desc: '「{name}」已经存在于你的 Skill Box 库(路径:{path}),如何处理?',
        overwrite: '覆盖',
        overwriteTip: '把已有的 {name} 替换成新下载的版本',
        rename: '另存为',
        renameTip: '保留旧版,新版本自动加 -2 / -3 后缀',
        cancel: '取消',
      },
    },
    // 2026-07-18 增:4 tab 共享的「目标分组」选择器 i18n
    targetGroup: {
      label: '导入到',
      root: '根分组(/)',
      hint: '所有 Tab 共享,默认根分组',
    },
    // 2026-07-01 增:从本地 zip / 文件夹导入面板
    // 2026-07-11 改:支持 zip / tar / tar.gz / tgz / tar.bz2 / tbz2 / tar.xz / txz
    local: {
      title: '从本地导入',
      desc: '从本地的文件夹或压缩包里读取 SKILL.md,直接落地到 Skill Box 的 store。',
      btnPickFolder: '选择文件夹',
      btnPickFolderTitle: '选一个本地目录,递归读取含 SKILL.md 的子目录',
      btnPickArchive: '选择压缩包',
      btnPickArchiveTitle: '选一个压缩包文件(zip / tar / tar.gz / tar.bz2 / tar.xz),解压后识别所有 SKILL.md',
      webNoFolder: 'Web 端不支持选文件夹,请用压缩包',
      webNoFolderHint: '请用下方"选择压缩包"按钮',
      importing: '导入中…',
      errNoPick: '未选择任何文件/目录',
      errNoSKILLMD: '未找到 SKILL.md 文件:目录或压缩包内必须存在 SKILL.md',
      errUnsupportedArchive: '暂不支持该压缩包格式,目前支持 zip / tar / tar.gz / tgz / tar.bz2 / tbz2 / tar.xz / txz',
      errImport: '导入失败:{msg}',
      okImport: '导入完成:成功 {ok} 个,失败 {failed} 个',
      statOk: '成功',
      statErr: '失败',
      statFound: '命中',
      resultTitle: '导入结果',
      btnAgain: '再导一次',
      btnDone: '完成',
    },
    // 2026-07-10 增:全局目录导入面板(从 ~/.agents/skills 列出候选,勾选批量导入)
    global: {
      title: '从全局目录导入',
      desc: '扫描 ~/.agents/skills 下所有 SKILL.md(Claude / Codex / Trae 等共享根),勾选要导入到 Skill Box 库的项目。',
      rootLabel: '扫描根',
      rootMissing: '目录不存在',
      rootMissingHint: '当前机器上没有 ~/.agents/skills,可先去任意支持 Agent Skills 标准的工具装几个 skill,再回来这里检索。',
      empty: '这个目录里还没有任何 skill',
      emptyHint: '可以先去 Claude / Codex / Trae 等工具里装一些 skill,然后再点"重新扫描"',
      loading: '扫描中…',
      loadFailed: '加载失败:{msg}',
      importOk: '导入完成:成功 {ok} 个,失败 {failed} 个',
      searchPlaceholder: '按名称、版本或描述搜索',
      selected: '已选 {sel} / {total}',
      selectAll: '全选当前',
      selectNone: '清空当前',
      btnRescan: '重新扫描',
      btnImport: '导入 {n} 个到 store',
      btnImportTitle: '把勾选的 skill 批量落地到 Skill Box 库',
      importing: '导入中…',
      colName: '名称',
      colVersion: '版本',
      colDesc: '描述',
      badgeImported: '已存在',
      tooltipImported: 'Skill Box 库里已经有同名 skill,跳过',
    },
    steps: {
      status: '查看状态',
      scan: '扫描 + 勾选',
      done: '完成',
    },
    phase1: {
      title: '工具 adapter 状态',
      total: '共 {n} 个',
      empty: '还没注册 adapter',
      colTool: '工具',
      colId: 'ID',
      colGlobalPath: '全局路径',
      colStatus: '状态',
      detected: '已检测到',
      missing: '未找到',
      lastScan: '上次扫描:',
      neverScanned: '从未',
      foundSuffix: '· 共发现 {n} 个技能',
      btnScan: '开始扫描',
      scanning: '扫描中…',
    },
    phase2: {
      title: '扫描结果',
      foundSuffix: '发现 {n} 个技能',
      empty: '这次扫描没找到任何技能',
      emptyHint: '可以点右上角"重新扫描",或先去工具里装一些 skill',
      selectAll: '全选当前',
      selectNone: '清空当前',
      selected: '已选 {sel} / {total}',
      btnBack: '返回上一步',
      btnImport: '导入 {n} 个到 store',
      importing: '导入中…',
      catUser: '用户技能',
      catSystem: '系统技能',
      catSystemHint: '系统级 skill(工具自带 / vendor curated / plugin 内建)只读展示,不能导入',
      catSectionDivider: '以下系统级 skill 不可勾选',
      tagExists: '已存在',
      disabledSystem: '系统级 skill 不能导入',
      disabledExists: '客户端 store 中已存在同名 skill,无法重复导入',
      disabledExclusive: '同名 skill 已被另一个工具勾选,请先取消',
    },
    phase3: {
      title: '导入完成',
      statOk: '成功',
      statErr: '失败',
      statTotal: '总计',
      btnAgain: '再扫一次',
      btnGoSkills: '去技能页查看',
    },
    errScan: '扫描失败: {msg}',
    errImport: '导入失败: {msg}',
    okImport: '导入完成: {ok} 成功 / {failed} 失败',
  },

  settings: {
    title: '设置',
    subtitle: '桌面端偏好(通知 / 全局快捷键 / 启动行为)。Web 端这部分是只读占位。',
    webOnlyHint: '桌面端偏好仅在桌面应用里可见。请用桌面端 / 系统托盘来打开设置。',

    general: {
      title: '通用偏好',
      subtitle: '通用设置',
      language: '界面语言',
      languageHint: '切换后立即生效,刷新或下次打开仍会保留',
      langZhCN: '简体中文',
      langEnUS: 'English',
    },

    // 2026-07-12 增:AI 模型(独立 card,Web / 桌面端均可见)。
    ai: {
      title: 'AI 模型',
      subtitle: '配置大模型用于翻译 / 优化技能等操作',
      btnNew: '新建',
      btnTest: '测试连接',
      btnTestTitle: '用当前配置向模型发起一次最小请求,验证连通性',
      listEmpty: '还没有配置任何模型,点右上"新建"开始',
      formNew: '新建模型',
      formEdit: '编辑模型',
      fieldName: '名称',
      fieldNameHint: '唯一标识,英文或中文',
      fieldKind: '厂商类型',
      // 2026-07-13 增:厂商类型下拉项的展示文案
      kindOpenAI: '官方',
      kindAnthropic: '官方',
      kindOpenAICompat: '兼容、硅基等',
      fieldBaseURL: '接口地址',
      fieldBaseURLHint: '留空走厂商默认;OpenAI 兼容型必须填',
      fieldModel: '模型名',
      fieldModelHint: '如 gpt-4o-mini / claude-3-5-sonnet / deepseek-chat',
      fieldApiKey: 'API Key',
      fieldApiKeyHint: '填入并保存到本地配置',
      fieldApiKeyEditHint: '留空 = 不修改;填入则覆盖',
      fieldPriority: '优先级',
      fieldEnabled: '启用',
      hasKey: '已配 Key',
      noKey: '未配 Key',
      badgeDisabled: '已停用',
      confirmDelete: '确定删除模型「{name}」?',
      errLoad: '加载失败:{msg}',
      errSave: '保存失败:{msg}',
      errDelete: '删除失败:{msg}',
      testOk: '测试成功',
    },

    desktop: {
      title: '桌面端偏好',
      subtitle: '需要重启桌面应用生效',
      startMinimized: '启动时最小化到托盘',
      startMinimizedHint: '勾选后,桌面应用启动时不再弹出主窗口,只在托盘留图标',
      notifyEnabled: '启用系统通知',
      notifyEnabledHint: '关闭后,"测试通知"按钮和托盘测试通知都不会发到通知中心',
      shortcutEnabled: '启用全局快捷键',
      shortcutEnabledHint: '关闭后,即使配了组合键也不响应(降级到只走菜单加速键)',
      globalHotkey: '全局快捷键组合',
      globalHotkeyHint: 'V1 仅支持 "Cmd+Shift+S"(macOS);其它组合在后端会拒绝注册',
      globalHotkeyPh: '如 Cmd+Shift+S',
    },

    // 2026-07-02 增:apply 模式(通用偏好,Web / 桌面端均可见)。
    applyMode: {
      title: '应用方式',
      subtitle: '技能应用到目标工具时的存在形式',
      copy: '复制',
      copyHint: '把 skill 源文件逐个拷贝到目标目录(占磁盘空间,源文件修改后需重新应用)',
      symlink: '软链接',
      symlinkHint: '在目标位置创建软链接指向源 skill(零占用,源文件修改后自动同步)',
      // 2026-07-02 改:两阶段 confirm —— 用户已点 segmented 触发切换,
      // 模式已写入 settings,现在单独问"是否把现有 {total} 条已应用 skill 重新落盘"。
      applyExistingToSymlinkConfirm: '检测到当前有 {total} 条已应用的 skill。\n\n是否同时把这些已应用 skill 改为软链接形式?\n\n选"是" = 重新落盘(后续修改源文件立即同步)\n选"否" = 保持原样(只影响未来新 apply)',
      applyExistingToCopyConfirm: '检测到当前有 {total} 条已应用的 skill。\n\n是否同时把这些已应用 skill 改为独立副本?\n\n选"是" = 重新落盘为目标端独立文件\n选"否" = 保持原样(只影响未来新 apply)',
      // 模式已切,迁移结果汇总
      modeChanged: '已切换到「{mode}」模式,新 apply 将按此方式落盘',
      modeChangedNoMigrate: '已切换到「{mode}」模式,{total} 条已应用的 skill 保持原样',
      switchMigrating: '正在迁移 {total} 条 skill...',
      switchSuccess: '迁移完成:成功 {ok} 条,跳过 {skipped} 条,失败 {failed} 条',
      switchFailedDetail: '迁移失败:\n{detail}',
      switchCancelled: '已取消',
    },

    testNotify: '测试通知',
    testNotifyHint: '向系统通知中心发一条测试横幅,验证授权 / 显示',
    btnTestNotify: '测试通知',

    testTitle: 'Skill Box',
    testBody: '这是一条测试通知 — 来自桌面端设置页',

    saved: '已保存',
    errSave: '保存失败: {msg}',
    errNotify: '通知失败: {msg}',
    notifyDisabled: '通知未启用,无法发送',
    notifySent: '通知已发送',

    prefsUnavailable: '偏好服务不可用(可能后端未启动或 prefs 存储未就绪)',
  },

  // 2026-07-01 增:工具元数据管理(对应 /api/skillbox/tools)
  tools: {
    title: '工具',
    subtitle: '共 {total} 个工具(系统 {system} + 用户 {user})。可启停 / 改字段 / 增删;改完点「重新加载」让 adapter 立刻生效。',
    searchPlaceholder: '按名称或 ID 过滤',
    filterAll: '全部',
    filterSystem: '系统',
    filterUser: '用户',
    btnNew: '新建',
    btnEdit: '编辑',
    btnReload: '重新加载',
    systemBadge: '系统',
    systemLocked: '系统工具不可删',
    pathCount: '{n} 个路径',
    // 2026-07-06 增:打开工具对应的 skills 目录按钮
    btnOpenSkillsDir: '打开技能目录',
    openNoPath: '该工具尚未配置 skills 目录',
    openFailed: '打开目录失败:{msg}',
    // 2026-07-07 增:reveal 失败时弹确认,允许用户自动创建并打开
    openCreateConfirm: '目录不存在,是否创建并打开?\n{path}',
    openCreateOk: '已创建并打开',
    openExistedOpen: '目录已存在,正在打开',
    openCreateFailed: '创建失败:{msg}',
    empty: '当前作用域下还没有工具',
    emptyHint: '点右上角「新建」开始;系统工具(9 个)由 seed 自动注入',
    loading: '加载中…',

    // maturity 三选一
    maturity: {
      stable: '稳定',
      experimental: '实验',
      deprecated: '已弃用',
    },

    // 字段 / 提示
    field: {
      toolId: '工具 ID',
      displayName: '展示名',
      mdiIcon: '图标 (mdi)',
      customIcon: '自定义图标',
      maturity: '成熟度',
      sortOrder: '排序',
      enabled: '启用',
      note: '备注',
    },
    hint: {
      toolId: 'canonical 短 ID,如 claude',
      toolIdLocked: 'tool_id 在创建后不可修改',
      displayName: 'UI 显示名',
      mdiIcon: '走 Iconify 在线解析,形如 mdi:robot-outline;清空时由「自定义图标」补位',
      customIcon: '上传本地 png/svg/ico 等(≤ 256KB)。优先级高于 mdi 图标。',
      note: '可选,内部备注',
    },
    formNewTitle: '新建工具',
    formEditTitle: '编辑「{name}」',
    formHint: '新建工具 is_system 强制为 false;Paths 用「覆盖式」语义,保存后整组替换。',
    btnUploadIcon: '上传自定义',
    btnClearIcon: '清除',
    uploadIconOk: '图标已上传',
    uploadIconFailed: '图标上传失败:{msg}',

    // paths 子表
    paths: {
      title: '技能路径',
      scope: '作用域',
      category: '类别',
      path: '路径',
      pathHint: '绝对路径,支持 ~/',
      pickFolder: '选择本地目录',
      scopeGlobal: '全局',
      scopeProject: '项目',
      categoryUser: '用户',
      categorySystem: '系统',
      // 2026-07-04 改:4 格固定布局,提示文案改成"清空即不生效"
      hint: '每个 (scope × category) 格子最多配置 1 条 path;留空 = 该档位不写入。',
    },

    // 反馈
    reloadedOk: '已重新加载',
    reloadFailed: '重新加载失败:{msg}',
    savedOk: '已保存',
    saveFailed: '保存失败:{msg}',
    enabledOk: '已启用',
    disabledOk: '已停用',
    toggleFailed: '启停切换失败:{msg}',
    deletedOk: '已删除',
    deleteFailed: '删除失败:{msg}',
    pickFolderFailed: '选择目录失败:{msg}',

    // 删除确认
    confirmDeleteTitle: '删除工具',
    confirmDeleteMsg: '确定删除「{name}」吗?该操作不可恢复,工具表行 + 全部 path 都会一并删除。',
  },

  // 2026-07-17 增:Git 同步面板(go-git 版本管理)文案。
  // 注意:这些 key 必须在 messages 根级别,因为 plainT 是按 dot path 查的。
  git: {
    title: 'Git 同步',
    showHistory: '查看历史',
    notInit: '未初始化',
    noCommits: '无提交',
    init: '初始化仓库',
    initTip: '点击下方按钮初始化本地仓库,之后技能改动会自动 commit + push 到远端。',
    remote: '远端',
    remoteMissing: '未配置',
    head: '当前',
    workingTree: '工作区',
    clean: '干净',
    dirty: '有改动',
    pending: '{n} 个 pending',
    pendingTip: '等待重试的 push 任务数',
    push: 'Push',
    discard: 'Discard',
    discardConfirm: '丢弃所有未提交改动?此操作不可撤销。',
    config: '配置远端',
    close: '收起',
    save: '保存',
    cancel: '取消',
    formRemoteURL: '远端 URL',
    formBranch: '分支',
    formToken: 'Token',
    formUserName: 'Author 名',
    formUserEmail: 'Author 邮箱',
    invalidUrl: '远端 URL 必须以 https:// 开头',
    noToken: '无 Token',
    // 2026-07-17 改:history.* 改成嵌套对象 — 之前平铺 historyXxx 跟
    // 模板 t('git.history.title') 点路径不匹配,显示成字面量。
    history: {
      title: '版本历史',
      initFirst: '请先在上方初始化 Git 仓库',
      empty: '暂无 commit 记录',
      emptySkill: '该技能暂无修改记录',
      diff: '变更对比',
      pickCommit: '点击左侧 commit 查看 diff',
      checkout: 'Reset 到此 commit',
      push: 'Push',
      pull: 'Pull',
      // 2026-07-17 增:VSCode 风格 — 展开 commit 后没有可显示的文件时提示
      noFiles: '本次提交未涉及具体文件',
      // 2026-07-17 增:diff modal 里的"所有文件"选项(列出完整 commit diff)
      allFiles: '所有文件',
    },
    // 2026-07-17 增:diff 暂不可用时给用户的提示 + 复制按钮
    copyCmd: '复制 git diff 命令',
    copied: '已复制到剪贴板:{cmd}',
    copyFailed: '复制失败:{msg}',
    checkoutConfirm: '确定要 reset 工作区到 commit {hash}?此操作会覆盖本地未保存的改动。',
  },
}

// 2026-07-09 改:市场域改成 iframe 嵌入,installDialog / pullDialog / sourcesSettings
// 等全部 key 已从 market 段移除,这里不再需要 alias 同步。

export default messages