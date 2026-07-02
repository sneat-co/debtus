import { test, expect } from '@playwright/test';

// Smoke scope (per decision): assert the app boots and routes resolve without
// crashing. Full "create split -> details" journey needs an authenticated
// session + seeded space/contacts (deferred — no auth seeding exists yet in
// this e2e project), so an unauthenticated login redirect is the expected
// path for every route below.

test('app boots and redirects unauthenticated user to login', async ({
  page,
}) => {
  const pageErrors: string[] = [];
  page.on('pageerror', (e) => pageErrors.push(String(e)));

  await page.goto('/');

  // Angular bootstrapped and rendered the app shell.
  await expect(page.locator('splitus-root')).toBeAttached();

  // Bootstrap + router + auth all work: root redirects to the login route.
  await page.waitForURL(/login/, { timeout: 20_000 });

  expect(pageErrors, `uncaught page errors:\n${pageErrors.join('\n')}`).toEqual(
    [],
  );
});

// splitus-space.routes.ts redirects the space root to 'splits' (not 'debts' —
// splitus has no debts route; this used to point at the old default before
// the splits home/list landed).
test('space-scoped splits route loads without crashing', async ({ page }) => {
  const pageErrors: string[] = [];
  page.on('pageerror', (e) => pageErrors.push(String(e)));

  // Lazy-loads the splitus space shell + splits (home/list) route.
  // Unauthenticated, this redirects to login; the assertion is that the app
  // handles it without throwing.
  await page.goto('/space/family/smoke-test-space/splits');

  await expect(page.locator('splitus-root')).toBeAttached();

  expect(pageErrors, `uncaught page errors:\n${pageErrors.join('\n')}`).toEqual(
    [],
  );
});

test('space-scoped new-split route loads without crashing', async ({
  page,
}) => {
  const pageErrors: string[] = [];
  page.on('pageerror', (e) => pageErrors.push(String(e)));

  await page.goto('/space/family/smoke-test-space/new-split');

  await expect(page.locator('splitus-root')).toBeAttached();

  expect(pageErrors, `uncaught page errors:\n${pageErrors.join('\n')}`).toEqual(
    [],
  );
});

test('space-scoped split details route loads without crashing', async ({
  page,
}) => {
  const pageErrors: string[] = [];
  page.on('pageerror', (e) => pageErrors.push(String(e)));

  await page.goto('/space/family/smoke-test-space/split/some-id');

  await expect(page.locator('splitus-root')).toBeAttached();

  expect(pageErrors, `uncaught page errors:\n${pageErrors.join('\n')}`).toEqual(
    [],
  );
});
