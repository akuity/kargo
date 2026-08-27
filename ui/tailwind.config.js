/** @type {import('tailwindcss').Config} */
module.exports = {
  mode: 'jit',
  // `dark:` keys off the data-theme attr on <html> (not `.dark`): CSS Modules
  // scopes class selectors, so `.dark` breaks inside *.module.less; attrs don't.
  // `variant` (not `selector`) because `selector` wraps the condition in
  // `:where()`, which adds no specificity -- a `dark:` utility then ties with its
  // own light counterpart and loses to any later stylesheet that redefines it.
  // UI extensions inject their own Tailwind build after ours, so a leaked
  // `.bg-gray-50` was beating `dark:bg-neutral-800`. `:is()` keeps the same
  // matching but carries specificity, and holds no class for CSS Modules to mangle.
  darkMode: ['variant', '&:is([data-theme="dark"], [data-theme="dark"] *)'],
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
