import { appRoutes } from './app.routes';

describe('appRoutes', () => {
  it('redirects the root path to login', () => {
    const root = appRoutes.find((r) => r.path === '');
    expect(root?.pathMatch).toBe('full');
    expect(root?.redirectTo).toBe('login');
  });

  it('mounts the space-scoped routes lazily', () => {
    const space = appRoutes.find(
      (r) => r.path === 'space/:spaceType/:spaceID',
    );
    expect(space).toBeDefined();
    expect(typeof space?.loadChildren).toBe('function');
  });
});
