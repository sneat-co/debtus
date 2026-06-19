import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { getStandardSneatProviders } from '@sneat/app';
import { splitusAppEnvironmentConfig } from '../environments/environment';
import { App } from './app';

describe('App', () => {
  beforeEach(() =>
    TestBed.configureTestingModule({
      imports: [App],
      // App extends BaseAppComponent, which injects auth/analytics/etc., so the
      // real app provider set is required for it to be created.
      providers: [
        ...getStandardSneatProviders(splitusAppEnvironmentConfig),
        provideRouter([]),
      ],
    }),
  );

  it('creates the root component', () => {
    const fixture = TestBed.createComponent(App);
    expect(fixture.componentInstance).toBeTruthy();
  });

  it('renders the Ionic app shell', () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    const host = fixture.nativeElement as HTMLElement;
    expect(host.querySelector('ion-app')).toBeTruthy();
  });
});
