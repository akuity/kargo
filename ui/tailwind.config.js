/** @type {import('tailwindcss').Config} */
module.exports = {
  mode: 'jit',
  // `dark:` keys off the data-theme attr on <html> (not `.dark`): CSS Modules
  // scopes class selectors, so `.dark` breaks inside *.module.less; attrs don't.
  darkMode: ['selector', '[data-theme="dark"]'],
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
