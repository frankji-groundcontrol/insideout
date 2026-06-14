/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    './src/components/**/*.{vue,js,ts,jsx,tsx}',
    './src/layouts/**/*.vue',
    './src/pages/**/*.vue',
    './src/plugins/**/*.{js,ts}',
    './src/features/**/*.{vue,js,ts}',
    './src/views/**/*.vue', // remove once views/ is fully emptied into pages/components
    './src/app.vue',
    './src/error.vue',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
