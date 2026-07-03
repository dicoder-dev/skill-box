// 全局图标映射表:把项目历史用的 mdi:xxx 字符串映射到 iconpark 组件名。
//
// 使用场景:
//   - <IconPark icon="mdi:loading" /> 内部查 MDI_TO_ICONPARK.loading → 'Loading'
//   - 业务代码里其他出现 mdi:xxx 的地方(路由 meta / store 默认值等),
//     仍可继续用 mdi 字符串,渲染时统一走本映射。
//
// 选图原则:
//   1. 优先 outline 主题(与原 mdi 一致)
//   2. 选最贴近的语义图形(若多个候选选较常用的,通常带 One 的更接近 mdi 风格)
//   3. 找不到完全对应的,选最相近语义,留个 (alias) 注释方便回查
//
// 维护:
//   - 加新映射直接在表里增一行,key 是 mdi 后半段,value 是 iconpark 组件名(PascalCase)。
//   - 不要重复 key;若需要给同一 mdi 多份候选,加注释说明。
//
// Why:
//   项目统一图标库规范要求所有图标走 iconpark,见 docs/agent/memory/fe-icon-library.md。

// 兜底:找不到对应时用 Help(问号,语义"未知/占位",比方块更可读)
export const NOT_FOUND_ICON = 'Help'

export const MDI_TO_ICONPARK = {
  // 关闭 / 取消 / 错误
  'close':                        'Close',
  'close-circle-outline':         'CloseOne',        // 圆环版关闭

  // 确认 / 成功
  'check':                        'Check',
  'check-circle':                 'CheckOne',        // 圆环对勾(填充)
  'check-circle-outline':         'CheckOne',        // 圆环对勾(描边)
  'check-decagram-outline':       'CheckCorrect',    // 稳定徽章

  // 信息 / 提示 / 警告
  'information':                  'Info',
  'information-outline':          'Info',
  'alert-circle-outline':         'Info',            // 警告
  'alert-octagon-outline':        'ErrorPrompt',     // 强提示(八角)
  'help-circle-outline':          'Help',

  // 搜索
  'magnify':                      'Search',
  'magnify-scan':                 'Find',            // 扫描型搜索
  'folder-search':                'FolderSearch',
  'folder-search-outline':        'FolderSearch',

  // 文件夹
  'folder-outline':               'Folder',
  'folder-open-outline':          'FolderOpen',
  'folder-open':                  'FolderOpen',
  'folder-plus-outline':          'FolderPlus',
  'folder-remove-outline':        'FolderMinus',
  'folder-account-outline':       'Folder',          // 用户文件夹(用通用)
  'folder-download-outline':      'FolderDownload',
  'folder-upload-outline':        'FolderUpload',
  'folder-zip-outline':           'Folder',
  'folder-multiple-outline':      'FileCabinet',     // 多文件柜
  'document-folder-outline':      'DocumentFolder',
  'inbox-outline':                'Inbox',

  // 文件
  'file-outline':                 'File',
  'file-document-outline':        'FileText',
  'text-box-outline':             'Text',            // 文本框
  'pencil-box-outline':           'EditOne',

  // 标签 / 分类
  'tag-outline':                  'Tag',
  'tag-off-outline':              'TagOne',          // 标签关闭态

  // 添加 / 移除
  'plus':                         'Plus',
  'plus-box':                     'Plus',
  'tray-arrow-down':              'Download',        // 下箭头托盘 → 下载

  // 刷新 / 同步 / 加载
  'refresh':                      'Refresh',
  'reload':                       'Reload',
  'sync':                         'Sync',
  'loading':                      'Loading',

  // 编辑 / 写入
  'pencil':                       'Pencil',
  'pencil-outline':               'Edit',
  'square-edit-outline':          'EditTwo',
  'rename-outline':               'EditName',
  'content-save':                 'Save',
  'content-save-outline':         'Save',
  'content-save-all-outline':     'Save',

  // 删除 / 危险
  'delete':                       'Delete',
  'delete-outline':               'DeleteOne',
  'trash-can':                    'Delete',
  'trash-can-outline':            'DeleteOne',

  // 设置 / 调整
  'cog-outline':                  'Setting',
  'tune-variant':                 'SettingTwo',
  'filter-variant':               'Filter',
  'view-list':                    'ListView',
  'format-list-bulleted':         'List',
  'format-list-numbered':         'ListNumbers',

  // 链接 / 外部
  'link-variant':                 'Link',
  'link-variant-off':             'LinkBreak',
  'open-in-new':                  'OpenOne',
  'external-link':                'LinkOut',

  // 视图 / 复制 / 眼睛
  'eye-outline':                  'View',
  'content-copy':                 'Copy',
  'content-copy-outline':         'Copy',

  // 上传 / 下载
  'upload':                       'Upload',
  'download':                     'Download',
  'download-outline':             'Download',
  'image-outline':                'Picture',
  'image-off-outline':            'CloseRemind',     // 图标占位
  'upload-outline':               'Upload',

  // 箭头
  'arrow-right':                  'ArrowRight',
  'arrow-left':                   'ArrowLeft',
  'arrow-up':                     'ArrowUp',
  'arrow-down':                   'ArrowDown',
  'chevron-right':                'ArrowRight',
  'chevron-left':                 'ArrowLeft',
  'chevron-down':                 'ArrowDown',
  'chevron-up':                   'ArrowUp',

  // 主题 / 明暗
  'weather-sunny':                'Sunny',
  'weather-night':                'Moon',

  // 通用 UI
  'menu':                         'List',             // 汉堡菜单
  'dots-vertical':                'More',
  'dots-horizontal':              'More',
  'bell-ring-outline':            'BellRing',
  'block-helper':                 'Block',
  'package-variant':              'Box',
  'package-variant-closed':       'Box',
  'rocket-launch-outline':        'Rocket',
  'test-tube':                    'TestTube',
  'flask-outline':                'Flask',
  'archive-arrow-down-outline':   'Inbox',            // 归档 → 收件

  // 工具 / 视图入口
  'tools':                        'Tool',
  'puzzle-outline':               'Puzzle',
  'shield-outline':               'Shield',
  'shield-check-outline':         'Shield',
  'cube-outline':                 'Cube',
  'desktop-classic':              'Computer',
  'monitor-dashboard':            'Dashboard',
  'cart-outline':                 'ShoppingCart',
  'book-open-variant':            'BookOpen',
  'console':                      'Terminal',
  'console-line':                 'Terminal',
  'chat-outline':                 'Comment',
  'cloud-off-outline':            'CloseWifi',
  'code-tags':                    'Code',
  'code-braces':                  'CodeBrackets',
  'robot':                        'Robot',
  'robot-outline':                'RobotOne',
  'leaf':                         'Leaf',
  'cursor-default-click-outline': 'Cursor',
  'script-text-outline':          'FileCode',
  'radio-tower':                  'Tower',
  'construction':                 'Tool',

  // 流程 / 状态
  'send':                         'Send',
  'stop':                         'Pause',
  'account-outline':              'User',
  'account-circle-outline':       'Avatar',
  'lock-outline':                 'Lock',
  'lock-open-outline':            'Unlock',
  'earth':                        'Earth',
  'globe-outline':                'Globe',
  'check-small':                  'CheckSmall',
  'circle-outline':               'Radio',
  'undo':                         'Undo',
  'redo':                         'Redo',
  'format-bold':                  'TextBold',
  'format-italic':                'TextItalic',
  'format-strikethrough':         'Strikethrough',
  'format-quote-close':           'Quote',
  'list-box-outline':             'List',
  'table-column-plus-after':      'Plus',
  'bug-outline':                  'Bug',
  'help':                         'Help',
}

export default MDI_TO_ICONPARK
