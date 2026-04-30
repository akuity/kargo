/** @type {import('tailwindcss').Config} */
module.exports = {
  mode: 'jit',
  content: ['./index.html', './src/**/*.{js,jsx,ts,tsx}'],
  theme: {
    extend: {}
  },
  plugins: [],
  corePlugins: {
    // https://github.com/ant-design/ant-design/issues/38794#issuecomment-1321806539
    preflight: false
  }
};
