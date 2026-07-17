// English language pack. Same structure as zh-CN.js.
//
// Punctuation / capitalization rules:
//   - Capitalize first word only in titles & button labels (sentence case)
//   - Keep code identifiers / values as-is (e.g. "codex", "global", "applied")
//   - Names of internal targets (file paths, API paths) NOT translated
//   - Placeholders use {name} syntax matching vue-i18n interpolation
//
// 2026-07-01: install → pull rename, installDialog becomes an alias of
// pullDialog (auto-synced at module bottom).

const messages = {
  app: {
    // 2026-07-11 changed: brand name back to 'Skill-Box' (was incorrectly
    // set to 'Q Boss' in v1). themeToggle copy moved into i18n so the
    // sidebar footer tooltip can consume it.
    // 2026-07-11 v11 changed: uppercase 'SKILL-BOX' + tracking for a
    // editorial brand-grade logo (Inter weight 800 + gradient + glow).
    // 2026-07-12 changed: 'SkillBox' (sentence case, no hyphen);
    // letter-spacing tightened to 0.5px since we are no longer uppercase.
    brand: 'SkillBox',
    closeSidebar: 'Close sidebar',
    openSidebar: 'Open sidebar',
    nav: {
      skills: { label: 'Skills' },
      projects: { label: 'Projects' },
      // 2026-07-01 new: tool metadata management
      tools: { label: 'Tools' },
      market: { label: 'Market' },
      onboarding: { label: 'Import skills' },
      settings: { label: 'Settings' },
    },
    backendOk: 'Backend connected',
    backendDown: 'Backend down',
    refreshStats: 'Refresh stats',
    toolsLabel: 'Tools',
    // 2026-07-11 new: topbar tooltip copy
    themeToggle: {
      toDark: 'Switch to dark mode',
      toLight: 'Switch to light mode',
    },
  },

  common: {
    cancel: 'Cancel',
    create: 'Create',
    save: 'Save',
    close: 'Close',
    delete: 'Delete',
    edit: 'Edit',
    apply: 'Apply',
    search: 'Search',
    refresh: 'Refresh',
    prev: 'Prev',
    next: 'Next',
    all: 'All',
    applyFilter: 'Apply filter',
    processing: 'Processing…',
    none: '—',
    dash: '—',
    confirm: 'Confirm',
    loading: 'Loading…',
    retry: 'Retry',
    optional: 'optional',
    count: '',
    notConfigured: 'not configured',
    // 2026-07-01 改:去 1000 上限后 total 数字不代表全网真数,误导。
    // 改为只显示"第 X / Y 页",Y 是 totalPages 由 size 推算出来。
    pageOf: 'Page {page} / {total} · {count} total',
    // 2026-07-01 增:市场专用分页 — 隐藏"共 N 条"误导数字,只显示页数。
    pageOfNoCount: 'Page {page} / {total}',
    totalCount: '{count} total',
    noData: 'No skills in this scope yet',
    noDataHint: 'Click "+ New" to start, or import from installed tools via Onboarding',
    deleted: 'Deleted {name}',
    openFailed: 'Open failed: {msg}',
  },

  skills: {
    title: 'Skills',
    subtitle: 'Browse / edit / test / apply / tag / rollback. The AI side panel rewrites frontmatter and body in one click.',
    scopeGlobal: 'Global',
    scopeProject: 'Project',
    searchPlaceholder: 'Filter by name',
    btnNew: '+ New Skill',
    btnAiOpen: 'Open AI',
    btnAiClose: 'Close AI',

    applyBar: {
      target: 'Apply target tool:',
      checkUpdates: 'Check updates',
      checking: 'Checking…',
      updatesAvailable: '{updates} / {total} updates available',
      allUpToDate: '{total} skills up to date',
      appliedOk: 'Applied {name}@{version} to {tool}',
      appliedPartial: 'Partial failure: {detail}',
      errDefault: 'Apply failed',
    },

    editor: {
      titleNew: 'New Skill',
      titleEdit: 'Edit Skill',
      name: 'Name',
      nameHint: 'short english id, e.g. review-pr',
      version: 'Version',
      versionHint: '0.1.0',
      scope: 'Scope',
      projectId: 'Project ID',
      description: 'Description',
      descriptionHint: 'min 10 chars',
      // 2026-06-27 new: small label before description on detail page (same style as triggers-label)
      descShort: 'Desc',
      triggers: 'Triggers',
      triggersHint: 'comma-separated',
      triggersHintPlaceholder: 'review pr, code review',
      // 2026-07-12 new: triggers are now optional; label badge + empty hint
      triggersOptional: 'optional',
      triggersEmptyHint: 'no triggers set — skill will not auto-fire on keywords',
      body: 'Body (Markdown, frontmatter auto-merged)',
      // 2026-06-26 new: scope refactor + apply tools
      scopeGlobal: 'Global',
      scopeProject: 'Project',
      scopeGlobalHint: 'visible to all projects',
      projectPick: 'select a project',
      applyTools: 'Apply to tools',
      applyToolsHint: 'auto-enable on selected tools',
      applyToolsNone: 'none selected, only stored in skillbox',
      applyToolsSelected: '{n} selected',
      errProjectRequired: 'please select a project first',
      applyAllSuccess: 'auto-enabled on {n} tools',
      applyPartialFailed: '{ok}/{total} tools enabled, failed: {fails}',
      // 2026-06-26 new: edit save sync to enabled locations
      syncAllSuccess: 'synced to {n} active locations',
      syncPartialFailed: '{ok}/{total} active locations synced',
      syncNone: '"{name}" saved, but not enabled anywhere yet — enable it from the scope section if needed',
      errNameEmpty: 'name is required',
      errVersionEmpty: 'version is required',
      errDescriptionEmpty: 'description is required',
      errDescShort: 'description must be at least 10 chars',
      errTriggersEmpty: 'at least one trigger is required',
      // 2026-07-13 new: frontmatter form + InlinePanel save/apply messages
      notReady: 'editor not ready yet, please try again in a moment',
      saveOk: 'Saved {name}',
      created: 'Created {name}@{version}',
      createdRefreshFailed: 'Created but refresh failed: {msg}',
      frontmatterDialogTitle: 'Edit / New frontmatter',
      triggerPlaceholder: 'trigger #{idx}',
      deleteTrigger: 'delete #{idx}',
      addTrigger: 'add trigger',
      descriptionRequiredPlaceholder: 'skill description (required)',
      licensePlaceholder: '(optional, e.g. MIT)',
      triggersList: 'triggers (list)',
      saving: 'Saving...',
    },

    // 2026-07-03 增:unified toast text for apply / batch partial-failure handling.
    apply: {
      allOk: 'Successfully applied to {n} tools',
      partialFailed: '{ok}/{total} tools applied successfully. Failures:\n{detail}',
    },

    applyHistory: {
      title: 'Recent apply history',
      count: '{count} entries',
      undone: 'Undo',
      undoing: 'Undoing…',
      applied: 'applied',
      rolledBack: 'rolled_back',
      failed: 'failed',
    },

    // 2026-07-17 删:tag block (retired along with ctag modal, replaced by go-git VersionHistoryModal).

    test: {
      title: 'Recent test result',
      errPrefix: 'Test failed:',
      passed: 'passed',
      failed: 'failed',
      errored: 'errored',
      skipped: 'skipped',
      confirmRun: 'Run test on skill "{name}@{version}"? (static + script + ai)',
    },

    list: {
      title: 'Skills',
      colName: 'Name',
      colVersion: 'Version',
      colSource: 'Source',
      colProject: 'Project',
      colUpdated: 'Updated',
      colActions: 'Actions',
      btnApply: 'Apply',
      applying: 'Applying…',
      btnTest: 'Test',
      testing: 'Testing…',
      btnEdit: 'Edit',
      btnTag: 'Tag',
      btnDelete: 'Delete',
      confirmDelete: 'Delete skill "{name}@{version}" ?',
      emptyTitle: 'No skills in this scope yet',
      emptyHint: 'Click "+ New Skill" to start, or import from installed tools via Onboarding',

      // left/right layout
      btnNewSkill: 'New',
      btnNewSkillTitle: 'New skill',
      btnImportSkill: 'Import',
      btnImportSkillTitle: 'Import from installed tools',
      searchTitle: 'Filter by name',
      selectToView: 'Pick a skill on the left to see details',
      // 2026-07-08: empty state — no skills yet, prompt user to create or import
      noSkillTitle: 'No skills yet',
      noSkillHint: 'Click "New" above to create your first skill, or "Import" to pull from installed tools',
      noSkillBtnCreate: 'New skill',
      noSkillBtnImport: 'Import from installed tools',
      noFilesHint: 'This skill has no renderable body',
      scopeLabel: 'Scope',
      scopeGlobalChip: 'Global',
      scopeProjectChip: 'Project',
      scopeToolsRow: 'Tools',
      scopeTargetsRow: 'Locations',
      scopeEmpty: 'This skill is not installed in any tool yet',
      scopeHitCount: '{n} locations',
      scopeSelectToolFirst: 'Pick a tool in the "Tools" row first, then choose a location',
      scopeForTool: 'Applies to {tool}',
      scopeToolSelected: 'Selected: {tool}',
      applyConfirmTitle: 'Enable location',
      applyConfirmMessage: 'Copy skill "{name}" to {tool} · {scope}?',
      applySuccess: 'Enabled: {path}',
      applyFailed: 'Failed to enable: {msg}',
      unapplyConfirmTitle: 'Disable location',
      unapplyConfirmMessage: 'Delete skill "{name}" from {tool} · {scope} (via apply/undo + PreSnapshot restore). Continue?',
      unapplySuccess: 'Disabled: {path}',
      unapplyFailed: 'Failed to disable: {msg}',
      appliedGlobal: '{tool} applied globally',
      applying: 'Enabling…',
      unapplying: 'Disabling…',
      tagsEmpty: 'No tags yet, click the icon above to create one',
      bodyEmpty: 'SKILL.md has no body yet',
      bodyTitle: 'Body',
      bodyEditing: 'Editing body (Markdown)',
      tooltipTest: 'Test',
      tooltipTag: 'Tag',
      tooltipOpenFolder: 'Open in folder',
      tooltipDelete: 'Delete',
      copied: 'Copied',
      openFailed: 'Open failed: {msg}',
      goOnboarding: 'Go import',

      // 2026-06-29: tree UI + context menu + drag
      treeEmpty: 'No skills yet. Click "New" to start.',
      treeRootHint: 'Right-click to create a group, or drag a skill here',
      ctxNewGroup: 'New group',
      ctxNewSubgroup: 'New sub-group',
      ctxDeleteGroup: 'Delete group',
      ctxRename: 'Rename',
      ctxOpenFolder: 'Open in folder',
      ctxCopyPath: 'Copy path',
      // 2026-07-17 改:tag modal retired, replaced by version history modal (go-git commit timeline).
      ctxVersion: 'View version history',
      ctxDelete: 'Delete',
      ctxMoveTo: 'Move to…',
      groupNamePrompt: 'Enter group name (use / for nested, e.g. frontend/react)',
      groupNamePromptSub: 'Enter sub-group name',
      groupInvalid: 'Invalid group name (no .., no leading/trailing /)',
      groupCreateFailed: 'Create group failed: {msg}',
      groupDeleteConfirm: 'Delete group "{name}"?',
      groupDeleteConfirmCascade: 'This group contains {n} skill(s); they will be deleted too. Also remove their copies from the 5 tool directories?',
      groupDeleteCascadeHint: 'Also remove copies from tool directories (5 tools × global / per-project)',
      groupDeleteFailed: 'Delete group failed: {msg}',
      // 2026-06-29 new: rename group
      groupRenamePrompt: 'Rename group "{name}"',
      groupRenameHint: 'Only the last segment is renamed; the parent path stays. Allowed chars: lowercase / digits / - / _',
      groupRenameOk: 'Renamed to "{name}"',
      groupRenameFailed: 'Rename failed: {msg}',
      groupRenameConflict: 'A group with the same name already exists at the same level',
      groupRenameNotFound: 'Source group no longer exists',
      skillDeleteConfirm: 'Delete skill "{name}"?',
      skillDeleteCascadeHint: 'Also remove copies from tool directories (5 tools × global / per-project)',
      skillDeleteFailed: 'Delete failed: {msg}',
      skillCascadeOk: 'Deleted "{name}" and {n} tool-dir copies',
      skillCascadePartial: 'Deleted "{name}"; {n} tool-dir cleanups failed: {detail}',
      skillCascadeSkipped: 'Deleted "{name}" (tool-dir copies left intact)',
      skillTagOpenFailed: 'Add tag failed: please select a skill first',
      skillOpenFolderOk: 'Opened in folder',
      skillOpenFolderFailed: 'Open failed: {msg}',
      moveFailed: 'Move failed: {msg}',
      moveTargetExists: 'Target already has a skill with the same name',
      moveSameGroup: 'Source and target are the same group, nothing to move',
      // 2026-06-29 new: dragging a group into its own descendant
      moveIntoDescendant: 'Cannot move a group into its own descendant',
      // 2026-06-29 new: drop-to-root visual hint + no-op toast
      dropToRoot: 'Drop to root',
      alreadyAtRoot: 'Already at root, nothing to move',
      // 2026-07-08 new: a skill can only be dropped onto a group or the root
      dropOnSkillNotAllowed: 'A skill can only be dropped onto a group or the root',
      loadingTree: 'Loading skills…',
    },

    // 2026-07-04 增:Skill file browser (skills top level, not inside list)
    fileBrowser: {
      open: 'Browse files',
      files: '{n} files',
      readOnly: 'Read only',
      readOnlyHint: 'SKILL.md is read-only here, edit in the main area',
      unsaved: 'You have unsaved changes. Close anyway?',
      unsavedShort: 'Unsaved',
      noFile: 'No file selected',
      pickOne: 'Pick a file from the left to preview',
      saved: 'Saved {path}',
      saveFailed: 'Save failed: {msg}',
      discard: 'Discard',
      save: 'Save',
      saving: 'Saving',
      binaryTitle: 'No online preview',
      binaryHint: 'Binary file (.{ext}) can be viewed in the folder',
      largeTitle: 'File too large for online preview',
      largeHint: 'File size {kb} KB, view it in the folder',
      openInFolder: 'Open in folder',
      // 2026-07-05 增:磁盘文件损坏提示(后端 corrupted_file 错误时弹)
      corruptedHint: 'Skill "{name}" has a corrupted SKILL.md file ({hint}). Please inspect the disk file or restore from backup.',
      // 2026-07-13 new: complete SkillFileInlinePanel + SkillScopePanel strings
      skillDirectory: 'skill directory',
      pickOneToBrowse: 'Pick a file to start browsing',
      renderError: 'Failed to load skill details',
      showOutline: 'Show outline',
      hideOutline: 'Hide outline',
      outline: 'Outline',
      viewFrontmatter: 'View frontmatter',
      ariaLabel: 'File browser',
      ctxNewFile: 'New file',
      ctxNewFolder: 'New folder',
      ctxRenameFile: 'Rename file',
      ctxRenameFolder: 'Rename folder',
      ctxDeleteFile: 'Delete file',
      ctxDeleteFolder: 'Delete folder',
      openInExplorer: 'Open in file explorer',
      newFileTitle: 'New file (will be created in the selected directory)',
      newFolderTitle: 'New folder (will be created in the selected directory)',
      renameFileTitle: 'Rename file',
      renameFolderTitle: 'Rename folder',
      deleteFileTitle: 'Delete file',
      deleteFileConfirm: 'Are you sure you want to delete "{name}"? This cannot be undone.',
      deleteFolderConfirm: 'Are you sure you want to delete folder "{name}"?',
      deleteFolderChildrenWarning: 'This folder contains {n} more files that will also be deleted. This cannot be undone.',
      newFilePlaceholder: 'file name (e.g. notes.md)',
      newFolderPlaceholder: 'folder name (e.g. examples)',
      modifiedTitle: 'File modified',
      discardPrompt: 'This file has been modified. How would you like to proceed before switching?',
      discardSaveHint: 'Save: write to disk before switching',
      discardDropHint: 'Discard: drop local edits and load the target skill/file',
      discardCancelHint: 'Cancel: stay on the current page and continue editing',
      discardChanges: 'Discard changes',
      saveChanges: 'Save changes',
      modifiedTitleDirty: 'File has been modified',
      incompleteFilesWarning: 'Warning: file list is empty, only the current file is being submitted. Other files may be lost — please wait until the directory finishes loading.',
      noSkillSelected: 'No skill selected',
      noFrontmatter: 'No frontmatter',
      modifiedState: '● Unsaved',
      modifiedShort: '● Unsaved',
      validation: {
        nameRequired: 'Name is required',
        invalidSeparator: 'Name cannot contain / or \\',
        invalidDotName: 'Name cannot be . or ..',
        invalidSKILL: 'SKILL.md is managed by the system and cannot be created directly',
        duplicateName: 'A file/folder with the same name already exists',
        duplicateFile: 'A file with the same name already exists in this folder',
      },
      createdFile: 'Created file "{name}"',
      createdDir: 'Created folder "{name}"',
      renamed: 'Renamed to "{name}"',
      deletedItem: 'Deleted "{name}"',
      sourceDirMissing: 'Cannot locate the disk directory (missing source_dir)',
      skillDirFolder: 'Skill directory',
      csvPreviewLimit: 'File has {total} rows, only first {preview} are previewed. Switch to edit mode for full content.',
      csvEmpty: 'Empty file or no parseable rows',
      officeParseFailed: 'Document parse failed',
      officeParseHint: 'The file may be corrupted or the binary content cannot be fully restored. Please use "Open in folder" to view in a native app.',
      emptyFolder: 'Empty folder',
      emptyShort: 'empty',
      largeShort: 'large',
      unsavedChanges: 'You have unsaved changes',
    },

    ai: {
      header: 'AI Assistant',
      clear: 'Clear',
      empty: 'Pick a preset first (optimize frontmatter / check description / polish body / find duplicates / security check), then ask.',
      hintNoProvider: 'No AI provider or built-in preset configured',
      pickFirst: 'Pick a preset from above first.',
      pickedDedupe: 'Paste the skill bodies you want to compare into the input (separate each with \\n\\n---\\n\\n), I will return overlap scores.',
      pickedPreset: 'Preset selected: 「{title}」. {description}\nPaste the context (optional) and extra requirements below, then hit Send.',
      roleUser: 'You',
      roleAssistant: 'AI',
      copy: 'Copy',
      inputPlaceholderHint: 'Additional notes (optional)',
      inputPlaceholderNoPreset: 'Pick a preset first',
      send: 'Send',
      stop: 'Stop',
      noExtraInput: '(no extra input, context only)',
      errorTag: '[error] {msg}',
    },

    // 2026-07-13 new: SkillScopePanel strings (migrated from LABEL_* constants).
    scope: {
      title: 'Skill scope',
      global: 'Global',
      // 2026-07-16 removed: projectPrefix (was used to compose "Project #N").
      // SkillScopePanel now reads project names directly from the projects data
      // table (configured on the Projects page) instead of stitching i18n text.
      empty: 'This skill has not been written to any tool/location yet',
      loading: 'Loading...',
      enable: 'Enable scope',
      disable: 'Disable scope',
      loadError: 'Failed to load scope',
      retry: 'Retry',
      enabled: 'enabled',
      disabled: 'disabled',
      partialFailed: 'Partial failure',
      enableFailed: 'Enable failed',
      disableFailed: 'Disable failed',
      enableSuccess: 'Enabled {tool} · {scope}',
      disableSuccess: 'Disabled {tool} · {scope}',
      disableConfirm: 'Remove skill "{name}" from {tool} · {scope}?',
      enableConfirm: 'Copy skill "{name}" to {tool} · {scope}?',
      toolCount: '{n} tools',
      toolCountShort: '{n} tools',
      globalEnabledTip: 'Global scope enabled',
      globalAgent: 'Global Agent Skill',
      globalAgentTip: 'Sync to ~/.agents/skills/ shared pool (readable by all tools)',
      globalAgentInfoTip: 'View global agent info',
      globalAgentFolderTip: 'Open ~/.agents/skills/ in file explorer',
      globalAgentInfoTitle: 'Global Agent Skill',
      globalAgentInfoDesc: 'After writing a skill to ~/.agents/skills/, any AI tool that declares this directory as its personal skills pool can read it automatically — no need to copy into each tool\'s directory.',
      globalAgentCompatibleToolsTitle: 'Tools that work with ~/.agents/skills/ (2026-07-12)',
      globalAgentSupported: 'Supported',
      globalAgentPartial: 'Partial',
      globalAgentEmpty: 'No compatible tools found.',
      globalAgentEnabled: 'Synced to ~/.agents/skills/{name}/',
      globalAgentDisabled: 'Removed from ~/.agents/skills/{name}/',
      globalAgentToggleFailed: 'Failed to toggle global agent: {msg}',
      globalAgentPathFailed: 'Cannot resolve ~/.agents/skills/ path',
      openFolderFailed: 'Failed to open folder: {msg}',
      globalAgentDirToast: 'Global Agent directory: {url}',
      toolNotes: {
        vscode: 'Documentation explicitly supports personal-level path',
        antigravity: 'Officially supported',
        claude: 'Project-level supported; personal level uses a different directory',
        codex: 'CLI follows the open standard',
        qwen: 'Officially supports the skill standard',
        cursor: 'Mainly uses a different directory; personal-level path not yet documented',
        opencode: 'Documentation does not explicitly support personal-level path',
        other: 'Path not mentioned in official docs',
      },
      dirToast: 'Global Agent directory: {path}',
    },

    // 2026-07-12 增:AI 弹窗(全局独立弹窗,替代旧 AIPanel 嵌入)。
    aiDialog: {
      title: 'AI Actions',
      subtitle: 'Common LLM-powered operations',
      btnOpen: 'AI',
      actionsTitle: 'Pick an action',
      actions: {
        translate: 'Translate Skill',
        translateDesc: 'Translate the full SKILL.md into a target language',
        optimize: 'Optimize Frontmatter',
        optimizeDesc: 'Rewrite name / description / triggers',
        comingSoon: 'Coming soon',
      },
      translate: {
        title: 'Translate Skill',
        desc: 'Translate the current SKILL.md into a target language. Frontmatter field names and code blocks are preserved.',
        targetLang: 'Target language',
        promptLabel: 'Original prompt',
        promptHint: 'Template with {target_lang} placeholder. Auto-replaced when you change the target language.',
        promptCopy: 'Copy prompt',
        promptCopied: 'Copied',
        // 已自定义提示词徽标(用户改过)
        promptCustomized: 'Modified',
        // 把自定义内容改回默认模板
        promptReset: 'Reset to default',
        // 2026-07-13 new: translation panel prompts
        promptEmpty: 'Prompt is empty — please fill it in or click "Reset to default"',
        startInstruction: 'Please follow the rules in the system prompt to start translating',
        // The "original prompt" template is kept in Chinese in both locales on purpose:
        // The user's target text is the translated skill body (which can be any language),
        // but the prompt itself stays consistent across locales to make round-trip
        // debugging and copying the prompt to other tools easier. The actual system
        // prompt sent to the LLM is injected by the backend preset, this is just a preview.
        promptTemplate:
`Translate the following Claude / Codex Skill "SKILL.md" into the target language:{target_lang}.

# Translation rules (strict)
1. Keep frontmatter field names (name / version / description / triggers) in English — do NOT translate field names.
2. If the description field value is a Chinese description, translate it into the target language.
3. Code blocks wrapped in triple backticks \`\`\` ... \`\`\` (commands, expected output, examples) MUST NOT be translated — leave them as-is.
4. Preserve Markdown structure (headings, lists, links, images, tables). Only translate the human text.
5. Style: professional, concise, technical-writing tone. No commentary, no preface, no afterword.
6. Output: ONLY the translated SKILL.md — no preamble, no explanation, no extra paragraphs.
7. If the source description field is already in English, still translate it if the target language differs from English.

# SKILL.md to translate
\`\`\`markdown
{skill_md}
\`\`\`

# Output (translate-only markdown body)
`,
        submit: 'Translate',
        submitting: 'Translating…',
        stop: 'Stop',
        resultTitle: 'Translation result',
        copyResult: 'Copy result',
        applyToEditor: 'Apply',
        applied: 'Applied',
        applyFailed: 'Apply failed: {msg}',
        noContext: '(No skill is selected, SKILL.md unavailable)',
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
      providerMissing: 'No AI model configured yet. Please set one up under Settings → AI Models first.',
      providerMissingTitle: 'AI model required',
    },

    // 2026-07-13 new: AI right panel (replaces the outline panel for AI assistance).
    // Lives next to the file viewer; chat-style UI with tag shortcuts (translate / review).
    aiPanel: {
      open: 'Open Agent Dog',
      close: 'Close Agent Dog',
      tagTranslate: 'Translate',
      tagReview: 'Review',
      inputPlaceholder: 'Ask Agent Dog… (Shift+Enter for newline)',
      send: 'Send',
      stop: 'Stop',
      closeBtn: 'Close',
      switchToOutline: 'Switch to outline',
      clearHistory: 'Clear chat',
      // 2026-07-14 added: chat history
      history: 'Chat history',
      // 2026-07-14 v2: rename "clear" to "new conversation" (archive current to history)
      newConv: 'New conversation',
      archiveFailed: 'Failed to save to history',
      deleteConv: 'Delete conversation',
      deleteFailed: 'Failed to delete',
      loadFailed: 'Failed to load',
      historyDialog: {
        title: 'Chat history',
        loading: 'Loading…',
        empty: 'No chat history yet',
      },
      roleYou: 'Me',
      roleAI: 'Agent Dog',
      emptyHint: "I'm agent dog — tap a tag below to get started.",
      noProvider: 'No AI provider configured yet. Please set one up under Settings → AI Models first.',
      noContent: 'The current file is empty — Agent Dog needs content to work with.',
      apply: 'Apply',
      reject: 'Reject',
      applied: 'Applied',
      rejected: 'Rejected',
      applyFailed: 'Apply failed: {msg}',
      reviewTitle: 'Review result',
      noIssues: 'No obvious issues found.',
      translateDialog: {
        title: 'Pick a target language',
        desc: 'After confirming, the translate prompt and the current document will be filled into the input box below — hit Send to run it.',
        confirm: 'Confirm',
        cancel: 'Cancel',
      },
      // 2026-07-17 改:translatePromptTemplate / reviewPromptTemplate /
      //   customPromptHint 三段含 "needs_apply": 字面 JSON 的字符串会触发
      //   vue-i18n ICU 解析器抛 "Invalid token in placeholder" 错误,导致
      //   SkillFileInlinePanel.onErrorCaptured 误显示 "技能详情加载出错"。
      //   模板已挪到 frontend/src/core/ai/promptTemplates.js。
      retrying: 'Regenerating… ({left} retries left)',
      parseFailed: 'AI response was malformed after 3 retries — shown as-is, no Apply button',
      truncated: 'AI output was too long, truncated',
      fullscreenEdit: 'Fullscreen edit',
      fullscreenEditTitle: 'Fullscreen editor',
      fullscreenSave: 'Save and return',
    },

    // 2026-07-13 new: AI apply / translate toasts (used by SkillsView.onAIApply)
    aiApply: {
      noSkill: 'Apply failed: no skill selected',
      empty: 'Apply failed: AI output is empty',
      applied: 'Applied (written back to {scope}/{name})',
      failed: 'Apply failed: {msg}',
    },

    // 2026-07-13 new: RichTextEditor (WYSIWYG editor for skill body) labels
    richText: {
      placeholder: 'Start typing...',
      heading1: 'Heading',
      heading2: 'Heading',
      heading3: 'Heading',
      bold: 'Bold',
      italic: 'Italic',
      strike: 'Strike',
      inlineCode: 'Inline code',
      bulletList: 'Bullet list',
      orderedList: 'Ordered list',
      blockquote: 'Blockquote',
      codeBlock: 'Code block',
      link: 'Link',
      unlink: 'Unlink',
      insertImageTip: 'Insert image (paste the URL)',
      imageUrl: 'Image URL',
      imageAlt: 'Alt text (optional)',
      insert: 'Insert',
      undo: 'Undo',
      redo: 'Redo',
    },

    // 2026-07-13 new: HelloWorld debug page
    helloWorld: {
      webNoLocalPort: 'Web mode (no local port)',
      dualDeployment: 'Dual deployment: desktop + web',
      deploymentMode: 'Deployment mode',
      appName: 'App name',
      localBackendPort: 'Local backend port',
      sameOrigin: 'empty, same origin',
      healthCheck: 'Health check',
      pingBackend: 'ping backend',
      enableDebugLog: 'Enable debug log',
      storeStatus: 'App status',
      httpScaffoldHint: 'API scaffold — continue writing per pattern, routes auto-register',
    },

    // 2026-07-13 new: AISettingsPanel provider type labels
    aiProvider: {
      official: 'Official',
      openAI: 'Official',
      anthropic: 'Official',
      openAICompat: 'Compatible / SiliconFlow etc.',
    },

    // 2026-07-16 removed: TreeNode global Agent badge text.
    // Per user feedback, the mdi:earth icon on the card is enough to convey
    // the "global Agent" semantic; an extra badge crowds the skill name.
    // globalAgentTip stays — used as the icon's hover title.
    treeNode: {
      globalAgentTip: 'This skill lives in the global agents directory and is read by all tools automatically',
      dropHere: 'Drop here',
    },
  },

  projects: {
    title: 'Projects',
    subtitle: 'Register project roots; later you can bind skills to a project scope to override global ones.',
    btnImport: 'Import Project',
    btnImportTitle: 'Import a project from a local folder',
    btnEdit: 'Edit',
    btnEditTitle: 'Edit project info',
    editFormTitle: 'Edit Project',
    btnPickAgain: 'Pick a different folder',
    btnCancel: 'Cancel',
    searchPlaceholder: 'Filter by name',
    formTitle: 'Import Project',
    formHint: 'Name and alias are auto-resolved from the selected folder. You can override them.',
    inspecting: 'Resolving…',
    name: 'Name',
    nameHint: 'display name, e.g. My App',
    alias: 'Alias',
    aliasHint: 'unique short id',
    rootPath: 'Root Path',
    rootPathHint: 'absolute path of project root',
    description: 'Description',
    descriptionHint: 'optional, project purpose',
    errRequired: 'name / alias / root_path are all required',
    listTitle: 'Projects',
    colId: 'ID',
    colName: 'Name',
    colAlias: 'Alias',
    colRootPath: 'Root Path',
    colDescription: 'Description',
    colActions: 'Actions',
    confirmDelete: 'Delete project #{id} ?',
    empty: 'No projects registered yet.',
    emptyHint: 'Click "Import Project" at the top right to start from a local folder',
    noTools: 'No tools applied to this project',
    openInFinder: 'Open in Finder',
    openFailed: 'Failed to open folder: {msg}',
    scannedAt: 'Scanned {time}',
    scanFailed: 'Scan failed',
    toolSkillsTitle: '{project} · {tool} skills ({count})',
    skillPath: 'Path',
    // Actions on each skill row inside the tool-skills modal
    skillActionReveal: 'Reveal this skill folder in file manager',
    skillActionDelete: 'Delete this skill folder (irreversible)',
    skillDeleteConfirm: 'Delete skill "{name}"? This will permanently remove the folder:\n{path}',
    skillDeleteFailed: 'Delete failed: {msg}',
    skillRevealFailed: 'Reveal failed: {msg}',
  },

  // 2026-07-09 改:MarketView 改回「卡片 + open in browser」方案。iframe 被 skillhub CORS 拒。
  //
  // 2026-07-10 改:
  //   - skillhub 国内源更名为 "skillhub-cn",i18n 同步换
  //   - GitHub tab 增 「known skill repos shortcut」
  //   - 装到 skill-box 按钮左侧增 「Paste」按钮
  market: {
    title: 'Marketplace',
    subtitle: 'Browse third-party skill catalogs (skillhub.cn / skills.sh / GitHub) and paste a detail page URL to install into skill-box.',
    btnOpenInBrowser: 'Open in browser',
    btnOpenInBrowserTip: 'Open {name} in system browser',
    // 2026-07-10: paste button (right of "Install" button)
    // 2026-07-10 改:rename "Paste" → "Paste & Install", paste ok auto-triggers install
    btnPaste: 'Paste & install',
    btnPasteTitle: 'Read clipboard text, paste into the input above and start installing immediately',
    btnPasteEmpty: 'Clipboard is empty, please copy a skill detail page URL first',
    btnPasteFailed: 'Failed to read clipboard: {msg}',
    // 2026-07-10: removed btnPasteSuccess(paste ok already auto-installs)
    cards: {
      skillhubDesc: 'A Skills community optimized for Chinese users, sorted by curated_score.',
      skillsshDesc: 'A Vercel-hosted leaderboard of AI agent skills, ranked by Hot (1H / change) metrics.',
      githubDesc: 'Agent skill repos on GitHub, install directly via raw content URL.',
    },
    // 2026-07-10: shortcut buttons for popular GitHub skill repos
    githubFamous: {
      title: 'Popular skill repos',
      desc: 'Below are well-known community-published agent skill repos on GitHub. Click a button to open in browser.',
      btnOpen: 'Open',
    },
    // 2026-07-09: input box one-click install
    guide: {
      title: 'How to install into skill-box',
      skillhub: {
        desc: 'Browse skillhub.cn and find a skill → copy the browser address bar URL → paste below.',
        examples: 'Examples',
        examplesList: [
          'https://skillhub.cn/skills/code-review',
          'https://skillhub.cn/skills/commit-msg',
        ],
      },
      skillssh: {
        desc: 'Browse skills.sh and find a skill → copy the detail page URL (or GitHub blob URL) → paste below.',
        examples: 'Examples',
        examplesList: [
          'https://skills.sh/anthropics/skills/pdf',
          'https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md',
        ],
      },
      github: {
        desc: 'Browse an agent skill repo on GitHub → find SKILL.md → copy the blob URL → paste below.',
        examples: 'Examples',
        examplesList: [
          'https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md',
          'https://github.com/anthropics/skills/blob/main/skills/code-review/SKILL.md',
        ],
      },
    },
    input: {
      label: 'Paste detail page URL',
      placeholderSkillhub: 'https://skillhub.cn/skills/{slug}',
      placeholderSkillssh: 'https://skills.sh/{owner}/{repo}/{skill} or GitHub blob URL',
      placeholderGithub: 'https://github.com/{owner}/{repo}/blob/{branch}/{path}/SKILL.md',
      // 2026-07-10: "Install to skill-box" → "Install" (page title already says skill-box)
      btnInstall: 'Install',
      btnInstalling: 'Installing…',
      errInvalidInput: 'Unrecognized input. Each tab only accepts its own market\'s detail URL.',
      errSource: 'Market source not found',
      // 2026-07-10: 422 — skill exists in upstream but SKILL.md is malformed
      // (典型:missing frontmatter / empty / zip not parseable).
      errSkillMalformed: 'Skill file is malformed ({msg}). Likely: author skipped the frontmatter, uploaded empty SKILL.md, or the zip is corrupt. This skill cannot be installed right now.',
      // 2026-07-10 改:404 细分 ——
      // - errSkillNotFound(slug typo / upstream removed)
      // - errSkillMalformed(走 422,文案见上)
      errSkillNotFound: 'Skill not found ({msg}). Likely: slug typo or upstream removed. Verify the URL and retry.',
      errPull: 'Download failed: {msg}',
      // 2026-07-09: separate timeout message
      errTimeout: 'Request timeout (slow backend download): {msg}. Try again, or download manually and import locally.',
      errGeneric: 'Install failed: {msg}',
      // 2026-07-13 new: market install extras
      clickToFill: 'Click to fill',
      rateLimitHint: 'Looks like rate limiting. Wait a few minutes and retry, or download manually in the browser and use "Local import".',
    },
    progress: {
      resolve: 'Resolving input…',
      download: 'Downloading skill…',
      extract: 'Extracting & validating…',
      write: 'Writing to skill-box store…',
      done: 'Installation complete',
      // 2026-07-09: 'fail' stage (keep progress on error, tell user which step failed)
      fail: 'Install failed',
      // 2026-07-09: sub-step hints under the progress bar
      hintResolve: 'Parsing URL, matching source type…',
      hintDownload: 'Downloading zip from GitHub codeload…',
      hintExtract: 'Extracting and validating SKILL.md…',
      hintWrite: 'Writing to local skill-box store…',
      hintDone: 'Installed, ready to jump',
      hintFailResolve: 'Failed during resolve, no request sent',
      hintFailDownload: 'Failed during download (zipball 404 / network / rate limit)',
      hintFailExtract: 'Failed during extract (SKILL.md parse error)',
      hintFailWrite: 'Failed during write to store (disk / permission)',
      hintFailUnknown: 'Unknown stage failure, see error below',
    },
    success: {
      msg: '✅ Installed {name} v{version}',
      goHome: 'View on home',
    },
    conflict: {
      title: 'Skill already exists',
      desc: 'A local skill named "{name}" already exists (v{existingVersion} @ {existingPath}). What would you like to do?',
      overwrite: 'Overwrite',
      overwriteTip: 'Replace the existing version with the new one',
      rename: 'Save as',
      renameTip: 'Append -2 / -3 suffix to keep both versions',
      cancel: 'Cancel',
    },
  },

  onboarding: {
    title: 'Import skills',
    subtitle: 'Scan skill directories of the 5 AI coding tools on this machine, pick which ones to import into the Skill Box store (global scope).',
    btnRescan: 'Rescan',
    btnRescanning: 'Scanning…',
    btnRescanTitle: 'Rescan all 5 adapters',
    // 2026-07-01: dialog top tab switcher
    // 2026-07-18: tab order changed + "Scan tools" renamed to "Tools" + new "From marketplace" tab first
    tabs: {
      market: 'From marketplace',
      scan: 'Tools',
      local: 'From local',
      // 2026-07-10: list candidates under ~/.agents/skills, batch import
      global: 'Global dir',
    },
    // 2026-07-18: marketplace import panel (3rd-party source list + URL input)
    market: {
      descSkillhub: 'SkillHub Chinese community market, fast in CN',
      descSkillssh: 'skills.sh international market, focused skill directory',
      descGithub: 'GitHub mainstream repos with SKILL.md detail pages',
      inputPlaceholder: 'Paste detail page URL (skillhub.cn / skills.sh / github.com)',
      btnImport: 'Import',
      btnImporting: 'Importing…',
      clear: 'Clear',
      errEmpty: 'Please paste a detail page URL',
      tip: 'Copy any detail page link from the sources above, paste into the input to import',
      // 2026-07-18: input examples (one per source)
      examplesLabel: 'Input examples (click to fill)',
      detectedSource: 'Detected {name}',
      gotoSite: 'Open {name} website',
      fillExample: 'Fill example',
    },
    // 2026-07-18: target group selector shared by all 4 tabs
    targetGroup: {
      label: 'Import to',
      root: 'Root group (/)',
      hint: 'Shared by all tabs, defaults to root',
    },
    // 2026-07-01: from-local import panel
    // 2026-07-11: extended to support zip / tar / tar.gz / tgz / tar.bz2 / tbz2 / tar.xz / txz
    local: {
      title: 'Import from local',
      desc: 'Pick a local folder or archive; we read SKILL.md and land them into the Skill Box store.',
      btnPickFolder: 'Pick folder',
      btnPickFolderTitle: 'Pick a local directory; recursively read sub-dirs with SKILL.md',
      btnPickArchive: 'Pick archive',
      btnPickArchiveTitle: 'Pick an archive (zip / tar / tar.gz / tar.bz2 / tar.xz); identify all SKILL.md after extraction',
      webNoFolder: 'Web mode does not support folder picker; use archive instead',
      webNoFolderHint: 'Please use "Pick archive" below',
      importing: 'Importing…',
      errNoPick: 'No file or folder selected',
      errNoSKILLMD: 'No SKILL.md found: the folder or archive must contain SKILL.md',
      errUnsupportedArchive: 'Unsupported archive format, currently supports zip / tar / tar.gz / tgz / tar.bz2 / tbz2 / tar.xz / txz',
      errImport: 'Import failed: {msg}',
      okImport: 'Imported: {ok} ok, {failed} failed',
      statOk: 'OK',
      statErr: 'Failed',
      statFound: 'Found',
      resultTitle: 'Import result',
      btnAgain: 'Import another',
      btnDone: 'Done',
    },
    // 2026-07-10: global-dir import panel (lists ~/.agents/skills candidates)
    global: {
      title: 'Import from global dir',
      desc: 'Scan ~/.agents/skills for SKILL.md files (shared root used by Claude / Codex / Trae). Tick the ones to import into the Skill Box store.',
      rootLabel: 'Scan root',
      rootMissing: 'Directory does not exist',
      rootMissingHint: 'No ~/.agents/skills on this machine yet. Install some skills via a tool that supports the Agent Skills standard, then come back and rescan.',
      empty: 'No skills found in this directory',
      emptyHint: 'Install some skills via Claude / Codex / Trae, then click "Rescan"',
      loading: 'Scanning…',
      loadFailed: 'Load failed: {msg}',
      importOk: 'Imported: {ok} ok, {failed} failed',
      searchPlaceholder: 'Search by name, version or description',
      selected: 'Selected {sel} / {total}',
      selectAll: 'Select all',
      selectNone: 'Clear',
      btnRescan: 'Rescan',
      btnImport: 'Import {n} into store',
      btnImportTitle: 'Land all ticked skills into the Skill Box store',
      importing: 'Importing…',
      colName: 'Name',
      colVersion: 'Version',
      colDesc: 'Description',
      badgeImported: 'Exists',
      tooltipImported: 'A skill with the same name already exists in the Skill Box store; skip',
    },
    steps: {
      status: 'Status',
      scan: 'Scan + select',
      done: 'Done',
    },
    phase1: {
      title: 'Tool adapter status',
      total: '{n} total',
      empty: 'No adapters registered yet',
      colTool: 'Tool',
      colId: 'ID',
      colGlobalPath: 'Global Path',
      colStatus: 'Status',
      detected: 'Detected',
      missing: 'Not found',
      lastScan: 'Last scan:',
      neverScanned: 'never',
      foundSuffix: '· {n} skills found',
      btnScan: 'Start scan',
      scanning: 'Scanning…',
    },
    phase2: {
      title: 'Scan result',
      foundSuffix: '{n} skills found',
      empty: 'No skills found this time.',
      emptyHint: 'Click "Rescan" in the top right, or install some skills first',
      selectAll: 'Select current',
      selectNone: 'Clear current',
      selected: '{sel} / {total} selected',
      btnBack: 'Back',
      btnImport: 'Import {n} into store',
      importing: 'Importing…',
      catUser: 'User skills',
      catSystem: 'System skills',
      catSystemHint: 'System-level skills (tool-built-in / vendor curated / plugin bundled) are read-only and cannot be imported',
      catSectionDivider: 'The following system-level skills cannot be selected',
      tagExists: 'Exists',
      disabledSystem: 'System-level skills cannot be imported',
      disabledExists: 'A skill with the same name already exists in the client store',
      disabledExclusive: 'The same skill is already selected from another tool — deselect first',
    },
    phase3: {
      title: 'Import complete',
      statOk: 'Succeeded',
      statErr: 'Failed',
      statTotal: 'Total',
      btnAgain: 'Scan again',
      btnGoSkills: 'View in Skills',
    },
    errScan: 'Scan failed: {msg}',
    errImport: 'Import failed: {msg}',
    okImport: 'Import complete: {ok} succeeded / {failed} failed',
  },

  settings: {
    title: 'Settings',
    subtitle: 'Desktop preferences (notifications / global shortcuts / startup). On Web, this section is a read-only placeholder.',
    webOnlyHint: 'Desktop preferences are only visible in the desktop app. Use the system tray to open Settings.',

    general: {
      title: 'General',
      subtitle: 'Common preferences',
      language: 'Display language',
      languageHint: 'Takes effect immediately. Your choice is remembered across reloads and sessions.',
      langZhCN: '简体中文',
      langEnUS: 'English',
    },

    // 2026-07-12 增:AI 模型(独立 card,Web / 桌面端均可见)。
    ai: {
      title: 'AI Models',
      subtitle: 'Configure LLM providers for actions like translate / optimize skill',
      btnNew: 'New',
      btnTest: 'Test connection',
      btnTestTitle: 'Send a minimal request with current config to verify connectivity',
      listEmpty: 'No models yet. Click "New" in the top right to add one.',
      formNew: 'New model',
      formEdit: 'Edit model',
      fieldName: 'Name',
      fieldNameHint: 'Unique identifier',
      fieldKind: 'Vendor',
      // 2026-07-13 new: provider kind option labels
      kindOpenAI: 'Official',
      kindAnthropic: 'Official',
      kindOpenAICompat: 'Compatible / SiliconFlow etc.',
      fieldBaseURL: 'Endpoint URL',
      fieldBaseURLHint: 'Leave empty to use vendor default; required for OpenAI-compatible',
      fieldModel: 'Model',
      fieldModelHint: 'e.g. gpt-4o-mini / claude-3-5-sonnet / deepseek-chat',
      fieldApiKey: 'API Key',
      fieldApiKeyHint: 'Saved locally upon save',
      fieldApiKeyEditHint: 'Leave empty = keep existing; fill to overwrite',
      fieldPriority: 'Priority',
      fieldEnabled: 'Enabled',
      hasKey: 'Key set',
      noKey: 'No key',
      badgeDisabled: 'Disabled',
      confirmDelete: 'Delete model "{name}"?',
      errLoad: 'Load failed: {msg}',
      errSave: 'Save failed: {msg}',
      errDelete: 'Delete failed: {msg}',
      testOk: 'Test passed',
    },

    desktop: {
      title: 'Desktop preferences',
      subtitle: 'Requires a desktop app restart to take effect',
      startMinimized: 'Start minimized to tray',
      startMinimizedHint: 'When enabled, the app starts hidden in the system tray without showing the main window',
      notifyEnabled: 'Enable system notifications',
      notifyEnabledHint: 'When off, "Test notification" button and tray test notifications are not delivered to the notification center',
      shortcutEnabled: 'Enable global shortcut',
      shortcutEnabledHint: 'When off, the registered combo does not respond (falls back to menu accelerator only)',
      globalHotkey: 'Global hotkey combo',
      globalHotkeyHint: 'V1 only supports "Cmd+Shift+S" on macOS; other combos are rejected by the backend',
      globalHotkeyPh: 'e.g. Cmd+Shift+S',
    },

    // 2026-07-02 add: apply mode (common preference, visible on both Web and Desktop).
    applyMode: {
      title: 'Apply mode',
      subtitle: 'How skills are placed on the target tool',
      copy: 'Copy',
      copyHint: 'Copy every source file to the target directory (uses disk space, requires re-apply after source changes)',
      symlink: 'Symlink',
      symlinkHint: 'Create a symlink at the target pointing to the source skill (zero disk usage, changes propagate automatically)',
      // 2026-07-02 update: two-stage confirm — mode is already written,
      // ask independently whether to re-apply the existing {total} skills.
      applyExistingToSymlinkConfirm: '{total} applied skills detected.\n\nRe-apply them as symlinks now?\n\nYes = re-place on disk (changes propagate automatically afterwards)\nNo  = keep current files (only future applies use the new mode)',
      applyExistingToCopyConfirm: '{total} applied skills detected.\n\nRe-apply them as independent copies now?\n\nYes = re-place on disk as standalone files\nNo  = keep current files (only future applies use the new mode)',
      modeChanged: 'Switched to "{mode}" — new applies will use this mode',
      modeChangedNoMigrate: 'Switched to "{mode}" — {total} existing applied skills were left untouched',
      switchMigrating: 'Migrating {total} skills...',
      switchSuccess: 'Migration done: {ok} ok, {skipped} skipped, {failed} failed',
      switchFailedDetail: 'Migration failures:\n{detail}',
      switchCancelled: 'Cancelled',
    },

    testNotify: 'Test notification',
    testNotifyHint: 'Send a test banner to the system notification center to verify authorization / display',
    btnTestNotify: 'Test notification',

    testTitle: 'Skill Box',
    testBody: 'This is a test notification — sent from the desktop settings page',

    saved: 'Saved',
    errSave: 'Save failed: {msg}',
    errNotify: 'Notification failed: {msg}',
    notifyDisabled: 'Notifications are disabled, cannot send',
    notifySent: 'Notification sent',

    prefsUnavailable: 'Preferences service unavailable (backend may not be running or prefs store not initialized)',
  },

  // 2026-07-01 new: tool metadata management (talks to /api/skillbox/tools)
  tools: {
    title: 'Tools',
    subtitle: '{total} tools total ({system} system + {user} user). Toggle / edit / add / delete; click "Reload" to make changes effective for adapters immediately.',
    searchPlaceholder: 'Filter by name or id',
    filterAll: 'All',
    filterSystem: 'System',
    filterUser: 'User',
    btnNew: 'New',
    btnEdit: 'Edit',
    btnReload: 'Reload',
    systemBadge: 'System',
    systemLocked: 'System tools cannot be deleted',
    pathCount: '{n} paths',
    // 2026-07-06: open the tool's skills directory in the system file manager
    btnOpenSkillsDir: 'Open skills directory',
    openNoPath: 'No skills directory configured for this tool',
    openFailed: 'Failed to open directory: {msg}',
    // 2026-07-07: confirm-create-and-open when reveal fails (directory missing)
    openCreateConfirm: 'Directory does not exist. Create it and open?\n{path}',
    openCreateOk: 'Created and opened',
    openExistedOpen: 'Directory already exists, opening',
    openCreateFailed: 'Failed to create: {msg}',
    empty: 'No tools in the current scope',
    emptyHint: 'Click "+ New" at the top right to start; the 9 system tools are auto-seeded',
    loading: 'Loading…',

    // maturity options
    maturity: {
      stable: 'Stable',
      experimental: 'Experimental',
      deprecated: 'Deprecated',
    },

    // fields / hints
    field: {
      toolId: 'Tool ID',
      displayName: 'Display name',
      mdiIcon: 'Icon (mdi)',
      customIcon: 'Custom icon',
      maturity: 'Maturity',
      sortOrder: 'Order',
      enabled: 'Enabled',
      note: 'Note',
    },
    hint: {
      toolId: 'canonical short id, e.g. claude',
      toolIdLocked: 'tool_id is immutable after creation',
      displayName: 'UI display name',
      mdiIcon: 'Resolved by Iconify at runtime, e.g. mdi:robot-outline. Cleared when "Custom icon" takes precedence.',
      customIcon: 'Upload a local png/svg/ico (≤ 256KB). Takes precedence over the mdi icon above.',
      note: 'optional, internal note',
    },
    formNewTitle: 'New tool',
    formEditTitle: 'Edit "{name}"',
    formHint: 'New tools are forced to is_system=false; Paths use overwrite semantics (saving replaces the whole group).',
    btnUploadIcon: 'Upload custom',
    btnClearIcon: 'Clear',
    uploadIconOk: 'Icon uploaded',
    uploadIconFailed: 'Icon upload failed: {msg}',

    // paths sub-table
    paths: {
      title: 'Skill Paths',
      scope: 'Scope',
      category: 'Category',
      path: 'Path',
      pathHint: 'absolute path, ~/ supported',
      pickFolder: 'Pick a folder',
      scopeGlobal: 'Global',
      scopeProject: 'Project',
      categoryUser: 'User',
      categorySystem: 'System',
      // 2026-07-04 改:4 fixed slots, hint explains "empty = skip"
      hint: 'Each (scope × category) slot allows at most 1 path; leave empty to skip that slot.',
    },

    // feedback
    reloadedOk: 'Reloaded',
    reloadFailed: 'Reload failed: {msg}',
    savedOk: 'Saved',
    saveFailed: 'Save failed: {msg}',
    enabledOk: 'Enabled',
    disabledOk: 'Disabled',
    toggleFailed: 'Toggle failed: {msg}',
    deletedOk: 'Deleted',
    deleteFailed: 'Delete failed: {msg}',
    pickFolderFailed: 'Pick folder failed: {msg}',

    // delete confirm
    confirmDeleteTitle: 'Delete tool',
    confirmDeleteMsg: 'Delete "{name}"? This cannot be undone; the tool row and all its paths will be removed.',
  },

  // 2026-07-17 add: Git sync panel (go-git version management) strings.
  git: {
    title: 'Git Sync',
    showHistory: 'View history',
    notInit: 'Not initialized',
    noCommits: 'No commits',
    init: 'Initialize repository',
    initTip: 'Click below to initialize a local repository. After that, skill changes are auto-committed and pushed to remote.',
    remote: 'Remote',
    remoteMissing: 'Not configured',
    head: 'Current',
    workingTree: 'Working tree',
    clean: 'Clean',
    dirty: 'Dirty',
    pending: '{n} pending',
    pendingTip: 'Push tasks waiting to retry',
    push: 'Push',
    discard: 'Discard',
    discardConfirm: 'Discard all uncommitted changes? This cannot be undone.',
    config: 'Configure remote',
    close: 'Collapse',
    save: 'Save',
    cancel: 'Cancel',
    formRemoteURL: 'Remote URL',
    formBranch: 'Branch',
    formToken: 'Token',
    formUserName: 'Author name',
    formUserEmail: 'Author email',
    invalidUrl: 'Remote URL must start with https://',
    noToken: 'No Token',
    // 2026-07-17 改:nested history object to match t('git.history.*') template calls.
    history: {
      title: 'Version history',
      initFirst: 'Please initialize the Git repository first',
      empty: 'No commits yet',
      emptySkill: 'No changes recorded for this skill',
      diff: 'Changes',
      pickCommit: 'Click a commit on the left to view diff',
      checkout: 'Reset to this commit',
      push: 'Push',
      pull: 'Pull',
      // 2026-07-17 增:VSCode-style expanded commit with no files
      noFiles: 'This commit touches no specific files',
      // 2026-07-17 增:diff modal "All files" pseudo-file entry
      allFiles: 'All files',
    },
    // 2026-07-17 增:diff unavailable fallback — copy git CLI command
    copyCmd: 'Copy git diff command',
    copied: 'Copied to clipboard: {cmd}',
    copyFailed: 'Copy failed: {msg}',
    checkoutConfirm: 'Reset working tree to commit {hash}? This will overwrite any unsaved local changes.',
  },
}

// 2026-07-09 改:市场域改成 iframe 嵌入,installDialog / pullDialog / sourcesSettings
// 等全部 key 已从 market 段移除,这里不再需要 alias 同步。

export default messages