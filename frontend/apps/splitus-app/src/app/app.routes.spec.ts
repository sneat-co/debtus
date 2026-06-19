import { appRoutes } from './app.routes';

describe('appRoutes', () => {
  it('serves an authenticated home component at the root path', () => {
    const root = appRoutes.find((r) => r.path === '');
    expect(root?.pathMatch).toBe('full');
    // Root must render a landing component, NOT redirect to login (which would
    // bounce signed-in users straight back to the login page).
    expect(root?.redirectTo).toBeUndefined();
    expect(typeof root?.loadComponent).toBe('function');
  });

  it('guards the root path so unauthenticated users go to login', () => {
    const root = appRoutes.find((r) => r.path === '');
    expect(root?.canActivate?.length).toBeGreaterThan(0);
    expect(typeof root?.data?.['authGuardPipe']).toBe('function');
  });

  it('mounts the space-scoped routes lazily', () => {
    const space = appRoutes.find(
      (r) => r.path === 'space/:spaceType/:spaceID',
    );
    expect(space).toBeDefined();
    expect(typeof space?.loadChildren).toBe('function');
  });
});
