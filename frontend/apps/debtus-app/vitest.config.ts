import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
    environmentOptions: { jsdom: { url: 'http://localhost/' } },
    // Ionic's icon lazy-loader tries to fetch SVGs in jsdom and throws benign
    // async "Invalid URL" errors when components with icons render in specs. All
    // assertions still pass; don't let these unhandled async errors fail the run.
    dangerouslyIgnoreUnhandledErrors: true,
    server: { deps: { inline: [/@ionic/, /ionicons/, /@sneat/] } },
    deps: {
      optimizer: {
        web: { include: ['@ionic/angular', '@ionic/core', 'ionicons'] },
      },
    },
  },
});
