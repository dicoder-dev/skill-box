/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  darkMode: 'class', // 启用 class 模式的暗黑模式
  theme: {
    extend: {
      colors: {
        sb: {
          // 侧边栏 - 浅色模式
          sidebar: '#f8fafc',
          'sidebar-hover': '#f1f5f9',
          'sidebar-active': '#2563eb',
          'sidebar-border': '#e2e8f0',
          'sidebar-muted': '#64748b',
          // 侧边栏 - 暗黑模式
          'sidebar-dark': '#0f172a',
          'sidebar-hover-dark': '#1e293b',
          'sidebar-active-dark': '#3b82f6',
          'sidebar-border-dark': '#334155',
          'sidebar-muted-dark': '#94a3b8',
          // 主区 - 浅色模式
          bg: '#f1f5f9',
          card: '#ffffff',
          border: '#e2e8f0',
          // 主区 - 暗黑模式
          'bg-dark': '#0f172a',
          'card-dark': '#1e293b',
          'border-dark': '#334155',
          // 文本 - 浅色模式
          text: '#1e293b',
          dim: '#64748b',
          faint: '#94a3b8',
          // 文本 - 暗黑模式
          'text-dark': '#f1f5f9',
          'dim-dark': '#94a3b8',
          'faint-dark': '#64748b',
          // 主色
          primary: '#2563eb',
          'primary-dark': '#3b82f6',
          'primary-dim': '#dbeafe',
          'primary-dim-dark': '#1e3a5f',
          'primary-hover': '#1d4ed8',
          // 状态色
          success: '#059669',
          'success-dim': '#d1fae5',
          'success-dim-dark': '#064e3b',
          warning: '#d97706',
          'warning-dim': '#fef3c7',
          'warning-dim-dark': '#78350f',
          danger: '#dc2626',
          'danger-dim': '#fee2e2',
          'danger-dim-dark': '#7f1d1d',
        },
      },
      boxShadow: {
        card: '0 1px 3px rgba(0, 0, 0, 0.06), 0 1px 2px rgba(0, 0, 0, 0.04)',
        'card-hover': '0 4px 6px rgba(0, 0, 0, 0.07), 0 2px 4px rgba(0, 0, 0, 0.04)',
        soft: '0 1px 2px rgba(0, 0, 0, 0.04)',
        'sidebar': '0 0 20px rgba(0, 0, 0, 0.08)',
      },
      borderRadius: {
        DEFAULT: '8px',
      },
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
  corePlugins: {
    preflight: false,
  },
}
