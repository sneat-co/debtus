import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { getStandardSneatProviders } from '@sneat/app';
import { splitusAppEnvironmentConfig } from '../../environments/environment';
import { MyProfilePageComponent } from './my-profile-page.component';

// Renders the profile page with the app's real provider set so the embedded
// sneat-user-auth-accounts / sneat-user-country DI chains resolve (catches a
// missing provider that would only surface at runtime as NG0201).
describe('MyProfilePageComponent', () => {
  beforeEach(() =>
    TestBed.configureTestingModule({
      imports: [MyProfilePageComponent],
      providers: [
        ...getStandardSneatProviders(splitusAppEnvironmentConfig),
        provideRouter([]),
      ],
    }),
  );

  it('renders the profile (all DI resolves, no NG0201)', () => {
    const fixture = TestBed.createComponent(MyProfilePageComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance).toBeTruthy();
    const host = fixture.nativeElement as HTMLElement;
    expect(host.querySelector('sneat-user-auth-accounts')).toBeTruthy();
  });
});
