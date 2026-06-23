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
          sidebar: '#ffffff',
          'sidebar-hover': '#f5f5f5',
          'sidebar-active': '#e5e5e5',
          'sidebar-border': '#e5e5e5',
          'sidebar-muted': '#737373',
          // 侧边栏 - 暗黑模式
          'sidebar-dark': '#0a0a0a',
          'sidebar-hover-dark': '#171717',
          'sidebar-active-dark': '#404040',
          'sidebar-border-dark': '#262626',
          'sidebar-muted-dark': '#a3a3a3',
          // 主区 - 浅色模式
          bg: '#ffffff',
          card: '#ffffff',
          subtle: '#fafafa',
          border: '#e5e5e5',
          // 主区 - 暗黑模式
          'bg-dark': '#0a0a0a',
          'card-dark': '#171717',
          'subtle-dark': '#141414',
          'border-dark': '#262626',
          // 文本 - 浅色模式
          text: '#171717',
          dim: '#525252',
          faint: '#a3a3a3',
          // 文本 - 暗黑模式
          'text-dark': '#fafafa',
          'dim-dark': '#a3a3a3',
          'faint-dark': '#525252',
          // 主色 - 纯黑/纯白强调
          primary: '#111111',
          'primary-dark': '#fafafa',
          'primary-dim': '#f5f5f5',
          'primary-dim-dark': '#262626',
          'primary-hover': '#000000',
          // 状态色 - 保持语义化但更克制
          success: '#15803d',
          'success-dim': '#f5f5f5',
          'success-dim-dark': '#262626',
          warning: '#a16207',
          'warning-dim': '#f5f5f5',
          'warning-dim-dark': '#262626',
          danger: '#b91c1c',
          'danger-dim': '#f5f5f5',
          'danger-dim-dark': '#262626',
        },
      },
      boxShadow: {
        card: '0 1px 2px rgba(0, 0, 0, 0.04)',
        'card-hover': '0 2px 4px rgba(0, 0, 0, 0.06)',
        soft: '0 1px 2px rgba(0, 0, 0, 0.04)',
        'sidebar': 'none',
      },
      borderRadius: {
        DEFAULT: '6px',
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
