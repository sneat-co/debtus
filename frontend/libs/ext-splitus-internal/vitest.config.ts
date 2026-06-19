import { createRequire } from 'node:module';
import { defineConfig } from 'vitest/config';

const require = createRequire(import.meta.url);

// `@ionic/angular` (its fesm bundles) imports from the bare directory
// `@ionic/core/components`, which Node's strict ESM resolver rejects because
// `@ionic/core` exposes no `./components` subpath export. Inlining the Ionic
// packages routes them through Vite's transform pipeline, where this plugin
// rewrites the bare directory specifier to its concrete `index.js`.
const ionicComponentsDirImportFix = {
  name: 'ionic-core-components-dir-import-fix',
  enforce: 'pre' as const,
  resolveId(id: string) {
    if (id === '@ionic/core/components') {
      return require.resolve('@ionic/core/components/index.js');
    }
    return null;
  },
};

export default defineConfig({
  plugins: [ionicComponentsDirImportFix],
  ssr: {
    noExternal: [/@ionic\//],
  },
  test: {
    server: { deps: { inline: [/@ionic/] } },
  },
});
