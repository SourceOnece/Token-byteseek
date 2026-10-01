// 边线透明度与 /70 等修饰符相乘，避免修饰符把低对比边框重新提亮。
const darkEdgeColor = (opacity) => ({ opacityValue = '1' }) =>
  `rgb(252 252 254 / calc(${opacity} * ${opacityValue}))`

// 深色边线独立于文字和表面色阶，border、divide、ring 共用同一强度。
const darkEdges = {
  400: darkEdgeColor(0.3),
  500: darkEdgeColor(0.125),
  600: darkEdgeColor(0.078),
  700: darkEdgeColor(0.04),
  800: darkEdgeColor(0.031),
  900: darkEdgeColor(0.02)
}

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    // 语义圆角档位:紧凑元素、控件、内容表面、桌面弹窗。
    // 数值统一由 style.css 的 :root --radius-* 变量定义,raw CSS 与工具类同源。
    // 旧尺度 key(sm/md/lg/xl…)已删除,check:ui 门禁阻止其复活。
    borderRadius: {
      none: '0px',
      compact: 'var(--radius-compact)',
      control: 'var(--radius-control)',
      surface: 'var(--radius-surface)',
      dialog: 'var(--radius-dialog)',
      full: '9999px'
    },
    extend: {
      borderColor: { dark: darkEdges },
      divideColor: { dark: darkEdges },
      ringColor: { dark: darkEdges },
      opacity: {
        6: '0.06',
        8: '0.08'
      },
      // 浮层层级语义档:数值唯一来源是 style.css :root 的 --z-* 变量,这里只做 var() 引用。
      // --z-tour(driver.js 外部约束)有意不暴露为工具类,防止业务代码依附第三方层级。
      zIndex: {
        'chart-tooltip': 'var(--z-chart-tooltip)',
        'sidebar-overlay': 'var(--z-sidebar-overlay)',
        sidebar: 'var(--z-sidebar)',
        header: 'var(--z-header)',
        modal: 'var(--z-modal)',
        'modal-nested': 'var(--z-modal-nested)',
        tooltip: 'var(--z-tooltip)',
        announcement: 'var(--z-announcement)',
        'announcement-raised': 'var(--z-announcement-raised)',
        'announcement-top': 'var(--z-announcement-top)',
        'menu-overlay': 'var(--z-menu-overlay)',
        toast: 'var(--z-toast)',
        'action-menu': 'var(--z-action-menu)',
        'teleport-tooltip': 'var(--z-teleport-tooltip)',
        'help-tooltip': 'var(--z-help-tooltip)',
        'teleport-dropdown': 'var(--z-teleport-dropdown)'
      },
      // 浮层面板最大高度三档,数值同样由 style.css :root 变量承载。
      maxHeight: {
        'menu-sm': 'var(--max-h-menu-sm)',
        menu: 'var(--max-h-menu)',
        panel: 'var(--max-h-panel)'
      },
      colors: {
        bh: {"red":"#E1251B","blue":"rgb(var(--theme-primary-600) / <alpha-value>)","yellow":"#FFCC00","ink":"#141414","paper":"#F4F0E6"},
        // 主色调 - Blue Archive 蓝白主题
        primary: {
  "50": "rgb(var(--theme-primary-50) / <alpha-value>)",
  "100": "rgb(var(--theme-primary-100) / <alpha-value>)",
  "200": "rgb(var(--theme-primary-200) / <alpha-value>)",
  "300": "rgb(var(--theme-primary-300) / <alpha-value>)",
  "400": "rgb(var(--theme-primary-400) / <alpha-value>)",
  "500": "rgb(var(--theme-primary-500) / <alpha-value>)",
  "600": "rgb(var(--theme-primary-600) / <alpha-value>)",
  "700": "rgb(var(--theme-primary-700) / <alpha-value>)",
  "800": "rgb(var(--theme-primary-800) / <alpha-value>)",
  "900": "rgb(var(--theme-primary-900) / <alpha-value>)",
  "950": "rgb(var(--theme-primary-950) / <alpha-value>)"
},
        // 辅助色 - 冰白到品牌深蓝
        accent: {
  "50": "rgb(var(--theme-accent-50) / <alpha-value>)",
  "100": "rgb(var(--theme-accent-100) / <alpha-value>)",
  "200": "rgb(var(--theme-accent-200) / <alpha-value>)",
  "300": "rgb(var(--theme-accent-300) / <alpha-value>)",
  "400": "rgb(var(--theme-accent-400) / <alpha-value>)",
  "500": "rgb(var(--theme-accent-500) / <alpha-value>)",
  "600": "rgb(var(--theme-accent-600) / <alpha-value>)",
  "700": "rgb(var(--theme-accent-700) / <alpha-value>)",
  "800": "rgb(var(--theme-accent-800) / <alpha-value>)",
  "900": "rgb(var(--theme-accent-900) / <alpha-value>)",
  "950": "rgb(var(--theme-accent-950) / <alpha-value>)"
},
        // 覆盖默认 gray/slate:Tailwind 默认值偏蓝(#1f2937/#0f172a 等),深色模式下会残留蓝调
        // 统一映射到中性 zinc 色相,与 dark 色阶同一体系
        gray: {
  "50": "rgb(var(--theme-gray-50) / <alpha-value>)",
  "100": "rgb(var(--theme-gray-100) / <alpha-value>)",
  "200": "rgb(var(--theme-gray-200) / <alpha-value>)",
  "300": "rgb(var(--theme-gray-300) / <alpha-value>)",
  "400": "rgb(var(--theme-gray-400) / <alpha-value>)",
  "500": "rgb(var(--theme-gray-500) / <alpha-value>)",
  "600": "rgb(var(--theme-gray-600) / <alpha-value>)",
  "700": "rgb(var(--theme-gray-700) / <alpha-value>)",
  "800": "rgb(var(--theme-gray-800) / <alpha-value>)",
  "900": "rgb(var(--theme-gray-900) / <alpha-value>)",
  "950": "rgb(var(--theme-gray-950) / <alpha-value>)"
},
        slate: {
  "50": "rgb(var(--theme-slate-50) / <alpha-value>)",
  "100": "rgb(var(--theme-slate-100) / <alpha-value>)",
  "200": "rgb(var(--theme-slate-200) / <alpha-value>)",
  "300": "rgb(var(--theme-slate-300) / <alpha-value>)",
  "400": "rgb(var(--theme-slate-400) / <alpha-value>)",
  "500": "rgb(var(--theme-slate-500) / <alpha-value>)",
  "600": "rgb(var(--theme-slate-600) / <alpha-value>)",
  "700": "rgb(var(--theme-slate-700) / <alpha-value>)",
  "800": "rgb(var(--theme-slate-800) / <alpha-value>)",
  "900": "rgb(var(--theme-slate-900) / <alpha-value>)",
  "950": "rgb(var(--theme-slate-950) / <alpha-value>)"
},
        // @project-doc docs/architecture/frontend_ui_conventions.md#dark_colors
        // 深色文字与表面色阶；边框强度由 darkEdges 单独定义。
        dark: {
  "50": "rgb(var(--theme-dark-50) / <alpha-value>)",
  "100": "rgb(var(--theme-dark-100) / <alpha-value>)",
  "200": "rgb(var(--theme-dark-200) / <alpha-value>)",
  "300": "rgb(var(--theme-dark-300) / <alpha-value>)",
  "400": "rgb(var(--theme-dark-400) / <alpha-value>)",
  "500": "rgb(var(--theme-dark-500) / <alpha-value>)",
  "600": "rgb(var(--theme-dark-600) / <alpha-value>)",
  "700": "rgb(var(--theme-dark-700) / <alpha-value>)",
  "800": "rgb(var(--theme-dark-800) / <alpha-value>)",
  "900": "rgb(var(--theme-dark-900) / <alpha-value>)",
  "950": "rgb(var(--theme-dark-950) / <alpha-value>)"
}
      },
      fontFamily: {
        display: ["\"Archivo Black\"","\"Plus Jakarta Sans Variable\"","system-ui","PingFang SC","Microsoft YaHei","sans-serif"],
        // 英文使用 OpenRouter 的开源字体，中文继续按现有系统字体顺序回退。
        sans: [
          '"Plus Jakarta Sans Variable"',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: [
          '"Geist Mono Variable"',
          'ui-monospace',
          'SFMono-Regular',
          'Menlo',
          'Monaco',
          'Consolas',
          'monospace'
        ]
      },
      boxShadow: {
  "DEFAULT": "var(--skin-shadow-DEFAULT)",
  "sm": "var(--skin-shadow-sm)",
  "md": "var(--skin-shadow-md)",
  "lg": "var(--skin-shadow-lg)",
  "xl": "var(--skin-shadow-xl)",
  "2xl": "var(--skin-shadow-2xl)",
  "glass": "var(--skin-shadow-glass)",
  "glass-sm": "var(--skin-shadow-glass-sm)",
  "glow": "var(--skin-shadow-glow)",
  "glow-lg": "var(--skin-shadow-glow-lg)",
  "card": "var(--skin-shadow-card)",
  "card-hover": "var(--skin-shadow-card-hover)",
  "inner-glow": "var(--skin-shadow-inner-glow)",
  "bh-sm": "var(--skin-shadow-bh-sm)",
  "bh": "var(--skin-shadow-bh)",
  "bh-lg": "var(--skin-shadow-bh-lg)",
  "bh-xl": "var(--skin-shadow-bh-xl)"
},
      backgroundImage: {
  "gradient-radial": "var(--skin-bg-gradient-radial)",
  "gradient-primary": "var(--skin-bg-gradient-primary)",
  "gradient-dark": "var(--skin-bg-gradient-dark)",
  "gradient-glass": "var(--skin-bg-gradient-glass)",
  "mesh-gradient": "var(--skin-bg-mesh-gradient)",
  "bh-stripe": "linear-gradient(90deg, #E1251B 0%, #E1251B 33.34%, #FFCC00 33.34%, #FFCC00 66.67%, #1450A3 66.67%, #1450A3 100%)"
},
      // 动效变量由 style.css 持有，工具类只负责引用。
      transitionDuration: {
        DEFAULT: 'var(--motion-fast)',
        fast: 'var(--motion-fast)',
        normal: 'var(--motion-normal)',
        layout: 'var(--motion-layout)',
      },
      transitionTimingFunction: {
        DEFAULT: 'var(--motion-ease)',
        standard: 'var(--motion-ease)',
        exit: 'var(--motion-ease-exit)',
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(0, 210, 255, 0.28)' },
          '100%': { boxShadow: '0 0 30px rgba(18, 167, 232, 0.4)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      }
    }
  },
  plugins: []
}
