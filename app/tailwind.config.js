/** @type {import('tailwindcss').Config} */
// Semantic tokens map onto the CSS custom properties in src/assets/tokens.css via
// `rgb(var(--x) / <alpha-value>)` so opacity modifiers keep working and a single
// `.dark` flip themes everything (设计系统交付物 §4). Prefixes fg-/stroke-/surface-
// avoid clobbering Tailwind's text/border/bg cores. Raw indigo-*/gray-* still resolve
// to Tailwind defaults, so adoption is an additive, non-breaking per-component codemod.
export default {
  darkMode: 'class',
  content: [
    './src/components/**/*.{vue,js,ts,jsx,tsx}',
    './src/layouts/**/*.vue',
    './src/pages/**/*.vue',
    './src/plugins/**/*.{js,ts}',
    './src/features/**/*.{vue,js,ts}',
    './src/app.vue',
    './src/error.vue'
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: 'rgb(var(--color-brand) / <alpha-value>)',
          hover: 'rgb(var(--color-brand-hover) / <alpha-value>)',
          subtle: 'rgb(var(--color-brand-subtle) / <alpha-value>)',
          ring: 'rgb(var(--color-brand-ring) / <alpha-value>)'
        },
        surface: {
          base: 'rgb(var(--color-surface-base) / <alpha-value>)',
          raised: 'rgb(var(--color-surface-raised) / <alpha-value>)',
          sunken: 'rgb(var(--color-surface-sunken) / <alpha-value>)',
          overlay: 'rgb(var(--color-surface-overlay) / <alpha-value>)'
        },
        stroke: {
          subtle: 'rgb(var(--color-stroke-subtle) / <alpha-value>)',
          strong: 'rgb(var(--color-stroke-strong) / <alpha-value>)',
          focus: 'rgb(var(--color-stroke-focus) / <alpha-value>)'
        },
        fg: {
          primary: 'rgb(var(--color-fg-primary) / <alpha-value>)',
          secondary: 'rgb(var(--color-fg-secondary) / <alpha-value>)',
          muted: 'rgb(var(--color-fg-muted) / <alpha-value>)',
          inverse: 'rgb(var(--color-fg-inverse) / <alpha-value>)',
          brand: 'rgb(var(--color-fg-brand) / <alpha-value>)',
          danger: 'rgb(var(--color-fg-danger) / <alpha-value>)'
        },
        status: {
          'neutral-bg': 'rgb(var(--color-status-neutral-bg) / <alpha-value>)',
          'neutral-fg': 'rgb(var(--color-status-neutral-fg) / <alpha-value>)',
          'info-bg': 'rgb(var(--color-status-info-bg) / <alpha-value>)',
          'info-fg': 'rgb(var(--color-status-info-fg) / <alpha-value>)',
          'warn-bg': 'rgb(var(--color-status-warn-bg) / <alpha-value>)',
          'warn-fg': 'rgb(var(--color-status-warn-fg) / <alpha-value>)',
          'success-bg': 'rgb(var(--color-status-success-bg) / <alpha-value>)',
          'success-fg': 'rgb(var(--color-status-success-fg) / <alpha-value>)',
          'danger-bg': 'rgb(var(--color-status-danger-bg) / <alpha-value>)',
          'danger-fg': 'rgb(var(--color-status-danger-fg) / <alpha-value>)'
        },
        // 国风留白 / Ink & Seal signature colors. `seal` is the vermilion 印泥 accent
        // (chops / status / links); `carve` is the paper color drawn on a seal or ink
        // fill; `btn`/`btn-fg` is the ink primary-action surface that inverts to paper
        // on dark. Use `bg-btn text-btn-fg` for primary buttons (the design keeps the
        // primary action ink, not vermilion — `seal`/`brand` stay the accent).
        seal: {
          DEFAULT: 'rgb(var(--color-seal) / <alpha-value>)',
          strong: 'rgb(var(--color-seal-strong) / <alpha-value>)',
          locked: 'rgb(var(--color-seal-locked) / <alpha-value>)'
        },
        carve: 'rgb(var(--color-carve) / <alpha-value>)',
        btn: {
          DEFAULT: 'rgb(var(--color-btn) / <alpha-value>)',
          fg: 'rgb(var(--color-btn-fg) / <alpha-value>)'
        }
      },
      borderRadius: {
        control: 'var(--radius-control)',
        card: 'var(--radius-card)',
        pill: 'var(--radius-pill)',
        hero: 'var(--radius-hero)'
      },
      boxShadow: {
        card: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
        popover:
          '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)',
        modal:
          '0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1)'
      },
      zIndex: {
        dropdown: '1000',
        sticky: '1010',
        overlay: '1020',
        modal: '1030',
        toast: '1040',
        tooltip: '1050'
      },
      // Alibaba PuHuiTi is the intended universal sans, named first so a visitor
      // who has it installed gets it — but it is NOT bundled (proprietary "free
      // to use", not redistributable; see style.css + README attribution).
      // Everyone else falls through to Noto Sans SC / system CJK. `serif` is the
      // Ink & Seal display face: Noto Serif SC (Song/Mincho) for headings + chop
      // labels, referenced via Google Fonts, falling back when it can't render.
      fontFamily: {
        sans: [
          '"Alibaba PuHuiTi"',
          '"Noto Sans SC"',
          'system-ui',
          '-apple-system',
          '"PingFang SC"',
          '"Microsoft YaHei"',
          'sans-serif'
        ],
        serif: [
          '"Noto Serif SC"',
          '"Songti SC"',
          '"Alibaba PuHuiTi"',
          'Georgia',
          'Cambria',
          '"Times New Roman"',
          'serif'
        ]
      }
    }
  },
  plugins: []
}
