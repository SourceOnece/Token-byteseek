/** @type {import('tailwindcss').Config} */

// ============================================================
// ByteSeek Bauhaus Theme
// 三原色（红/蓝/黄）+ 黑 + 纸白 / 直角 / 硬阴影 / 几何
// 与 docs.byteseek.ai 文档站同源的设计语言
// ============================================================

const BH = {
  red: '#E1251B',
  blue: '#1450A3',
  yellow: '#FFCC00',
  ink: '#141414',
  paper: '#F4F0E6'
}

export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    // 包豪斯：一切皆直角。仅保留 full 用于圆形（圆也是包豪斯的基本形）。
    borderRadius: {
      none: '0px',
      sm: '0px',
      DEFAULT: '0px',
      md: '0px',
      lg: '0px',
      xl: '0px',
      '2xl': '0px',
      '3xl': '0px',
      '4xl': '0px',
      full: '9999px',
      compact: '0px',
      control: '0px',
      surface: '0px',
      dialog: '0px'
    },
    extend: {
      colors: {
        // 包豪斯命名色，供视图直接使用
        bh: {
          red: BH.red,
          blue: BH.blue,
          yellow: BH.yellow,
          ink: BH.ink,
          paper: BH.paper
        },
        // 主色调 - 包豪斯蓝（主要操作、链接、激活态）
        primary: {
          50: '#F0F4FA',
          100: '#DFE8F5',
          200: '#B9CCE8',
          300: '#8BAAD8',
          400: '#5581C2',
          500: '#1450A3',
          600: '#1450A3',
          700: '#0F3D7D',
          800: '#0B2D5C',
          900: '#081F40',
          950: '#051225'
        },
        // 辅助色 - 包豪斯红（强调、品牌）
        accent: {
          50: '#FCF0EF',
          100: '#F9DEDC',
          200: '#F2B7B3',
          300: '#EB8F89',
          400: '#E55A51',
          500: '#E1251B',
          600: '#C21F16',
          700: '#9E1912',
          800: '#7A130E',
          900: '#560D0A',
          950: '#3B0906'
        },
        // 中性色 - 暖调纸灰，向纸色靠拢
        gray: {
          50: '#FAF8F2',
          100: '#F4F0E6',
          200: '#E6E1D3',
          300: '#D2CCBB',
          400: '#A39E8F',
          500: '#736F63',
          600: '#57534A',
          700: '#403D36',
          800: '#2B2925',
          900: '#1C1A16',
          950: '#141414'
        },
        slate: {
          50: '#FAF8F2',
          100: '#F4F0E6',
          200: '#E6E1D3',
          300: '#D2CCBB',
          400: '#A39E8F',
          500: '#736F63',
          600: '#57534A',
          700: '#403D36',
          800: '#2B2925',
          900: '#1C1A16',
          950: '#141414'
        },
        // 深色模式背景 - 暖黑（墨色纸背面），950 略亮于 900 作提升面
        dark: {
          50: '#F4F0E6',
          100: '#EAE5D8',
          200: '#CFC9B8',
          300: '#9E998B',
          400: '#736F63',
          500: '#55524A',
          600: '#3A3831',
          700: '#2F2D27',
          800: '#26231D',
          900: '#1C1A16',
          950: '#211F1A'
        }
      },
      fontFamily: {
        // 正文继续用 Plus Jakarta Sans（几何人文无衬线），中文回退系统字体
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
        // 展示字体：Archivo Black —— 厚重几何，海报级标题专用
        display: [
          '"Archivo Black"',
          '"Plus Jakarta Sans Variable"',
          'system-ui',
          'PingFang SC',
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
        // 包豪斯硬阴影：无模糊、纯位移。深色模式由 style.css 统一换成纸色阴影。
        // 所有标准阴影统一使用仪表盘卡片的 4px 硬阴影，避免页面层级各自为政。
        DEFAULT: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        sm: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        md: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        lg: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        xl: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        '2xl': `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        glass: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        'glass-sm': `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        glow: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        'glow-lg': `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        card: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        'card-hover': `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        'inner-glow': `inset 0 0 0 2px var(--bh-shadow-ink, #141414)`,
        'bh-sm': `3px 3px 0 0 var(--bh-shadow-ink, #141414)`,
        bh: `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        'bh-lg': `4px 4px 0 0 var(--bh-shadow-ink, #141414)`,
        'bh-xl': `4px 4px 0 0 var(--bh-shadow-ink, #141414)`
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        // 包豪斯不用渐变——映射为纯色块
        'gradient-primary': `linear-gradient(0deg, ${BH.blue} 0%, ${BH.blue} 100%)`,
        'gradient-dark': 'linear-gradient(0deg, #1C1A16 0%, #1C1A16 100%)',
        'gradient-glass': 'linear-gradient(0deg, rgba(255,255,255,0) 0%, rgba(255,255,255,0) 100%)',
        'mesh-gradient': 'none',
        // 三原色条纹（红/黄/蓝），用于顶栏或分隔装饰
        'bh-stripe': `linear-gradient(90deg, ${BH.red} 0%, ${BH.red} 33.34%, ${BH.yellow} 33.34%, ${BH.yellow} 66.67%, ${BH.blue} 66.67%, ${BH.blue} 100%)`
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate',
        'bh-rise': 'bhRise 0.6s ease-out both',
        'bh-float': 'bhFloat 5s ease-in-out infinite',
        'bh-bob': 'bhBob 3.2s ease-in-out infinite',
        'bh-spin-slow': 'bhSpin 14s linear infinite',
        'bh-marquee': 'bhMarquee 22s linear infinite'
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
          '0%': { boxShadow: `4px 4px 0 0 var(--bh-yellow, #FFCC00)` },
          '100%': { boxShadow: `6px 6px 0 0 var(--bh-yellow, #FFCC00)` }
        },
        bhRise: {
          from: { opacity: '0', transform: 'translateY(26px)' },
          to: { opacity: '1', transform: 'translateY(0)' }
        },
        bhFloat: {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-22px)' }
        },
        bhBob: {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-10px)' }
        },
        bhSpin: {
          from: { transform: 'rotate(0deg)' },
          to: { transform: 'rotate(360deg)' }
        },
        bhMarquee: {
          from: { transform: 'translateX(0)' },
          to: { transform: 'translateX(-50%)' }
        }
      },
      backdropBlur: {
        xs: '0px'
      }
    }
  },
  plugins: []
}
