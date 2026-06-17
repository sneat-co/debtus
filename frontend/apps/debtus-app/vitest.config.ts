import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    server: { deps: { inline: [/@ionic/, /ionicons/, /@sneat/] } },
    deps: {
      optimizer: {
        web: { include: ['@ionic/angular', '@ionic/core', 'ionicons'] },
      },
    },
  },
});
