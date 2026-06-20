import nx from '@nx/eslint-plugin';

export default [
  ...nx.configs['flat/base'],
  ...nx.configs['flat/typescript'],
  ...nx.configs['flat/javascript'],
  {
    ignores: [
      '**/dist',
      '**/vite.config.*.timestamp*',
      '**/vitest.config.*.timestamp*',
    ],
  },
  {
    files: ['**/*.ts', '**/*.tsx', '**/*.js', '**/*.jsx'],
    rules: {
      '@nx/enforce-module-boundaries': [
        'error',
        {
          enforceBuildableLibDependency: true,
          allow: ['^.*/eslint(\\.base)?\\.config\\.[cm]?[jt]s$'],
          depConstraints: [
            {
              sourceTag: 'scope:debtus',
              onlyDependOnLibsWithTags: ['scope:debtus'],
            },
            {
              sourceTag: 'scope:splitus',
              onlyDependOnLibsWithTags: ['scope:splitus'],
            },
            {
              sourceTag: 'type:contract',
              onlyDependOnLibsWithTags: ['type:contract', 'scope:foundation'],
            },
            {
              sourceTag: 'type:shared',
              onlyDependOnLibsWithTags: [
                'type:contract',
                'type:shared',
                'scope:foundation',
                'scope:debtus',
                'scope:splitus',
              ],
            },
            {
              sourceTag: 'type:internal',
              onlyDependOnLibsWithTags: [
                'type:contract',
                'type:shared',
                'type:internal',
                'scope:foundation',
                'scope:debtus',
                'scope:splitus',
              ],
            },
            {
              // The app is the composition root: it may consume every tier,
              // including type:internal (to wire provider factories at bootstrap).
              sourceTag: 'type:app',
              onlyDependOnLibsWithTags: [
                'type:lib',
                'type:contract',
                'type:shared',
                'type:internal',
              ],
            },
            {
              sourceTag: 'type:e2e',
              onlyDependOnLibsWithTags: ['type:app', 'type:lib'],
            },
            {
              sourceTag: 'type:lib',
              onlyDependOnLibsWithTags: ['type:lib', 'type:contract'],
            },
          ],
        },
      ],
    },
  },
  {
    files: [
      '**/*.ts',
      '**/*.tsx',
      '**/*.cts',
      '**/*.mts',
      '**/*.js',
      '**/*.jsx',
      '**/*.cjs',
      '**/*.mjs',
    ],
    // Override or add rules here
    rules: {},
  },
];
