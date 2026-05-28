/** @type {import('dependency-cruiser').IConfiguration} */
module.exports = {
  forbidden: [
    {
      name: 'no-circular',
      severity: 'error',
      from: {},
      to: { circular: true },
    },
    {
      name: 'no-api-to-web',
      severity: 'error',
      from: { path: '^apps/api' },
      to: { path: '^apps/web' },
    },
    {
      name: 'no-apps-to-internal-api-httpserver',
      severity: 'warn',
      from: { path: '^apps/(web|ai)' },
      to: { path: '^apps/api/internal/httpserver' },
    },
  ],
  options: {
    tsConfig: {
      fileName: './apps/web/tsconfig.json',
    },
    enhancedResolveOptions: {
      extensions: ['.ts', '.tsx', '.js', '.jsx', '.mjs', '.cjs', '.json'],
    },
    reporterOptions: {
      dot: {
        collapsePattern: '^(apps|packages)/[^/]+',
      },
    },
    exclude: {
      path: ['node_modules', '\\.turbo', 'dist', 'build', 'coverage'],
    },
  },
};
