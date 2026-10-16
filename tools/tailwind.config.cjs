// Build-time only: `make css` compiles web/assets/tailwind.css from the classes used in web/.
module.exports = {
  darkMode: 'class',
  content: ['./web/index.html', './web/assets/*.mjs'],
  theme: { extend: {} },
};
