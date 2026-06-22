/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        sb: {
          // sidebar 深色
          sidebar: '#1f2937',
          'sidebar-hover': 'rgba(255,255,255,0.06)',
          'sidebar-active': '#2563eb',
          'sidebar-border': '#374151',
          'sidebar-muted': '#9ca3af',
          // 主区
          bg: '#f5f7fa',
          card: '#ffffff',
          border: '#e5e7eb',
          // 文本
          text: '#1f2937',
          dim: '#6b7280',
          faint: '#9ca3af',
          // 状态
          primary: '#2563eb',
          'primary-dim': '#dbeafe',
          success: '#059669',
          'success-dim': '#d1fae5',
          warning: '#d97706',
          'warning-dim': '#fef3c7',
          danger: '#dc2626',
          'danger-dim': '#fee2e2',
        },
      },
      boxShadow: {
        card: '0 1px 3px rgba(0, 0, 0, 0.06)',
        soft: '0 1px 2px rgba(0, 0, 0, 0.04)',
      },
    },
  },
  plugins: [],
  corePlugins: {
    // 不用 preflight,避免和现有 button/input 默认样式冲突
    preflight: false,
  },
}
