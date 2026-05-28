module.exports = {
  fileExtensions: ['ts', 'tsx', 'js', 'jsx', 'mjs', 'cjs'],
  excludeRegExp: [
    '^node_modules',
    '^dist',
    '^build',
    '\\.test\\.',
    '\\.spec\\.',
  ],
  tsConfig: './apps/web/tsconfig.json',
  detectiveOptions: {
    es6: { mixedImports: true },
    ts: { skipTypeImports: true },
  },
};
