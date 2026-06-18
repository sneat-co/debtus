import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { getStandardSneatProviders } from '@sneat/app';
import { SneatUserService } from '@sneat/auth-core';
import { BehaviorSubject } from 'rxjs';
import { debtusAppEnvironmentConfig } from '../../environments/environment';
import { DebtusHomePageComponent } from './debtus-home-page.component';

// Uses the app's real provider set (getStandardSneatProviders) so the full DI
// chain of the embedded SpacesCardComponent is exercised. This is the test that
// catches a missing provider (e.g. SpaceService, UserRequiredFieldsService)
// which only surfaces at runtime as NG0201, not at build time.
describe('DebtusHomePageComponent', () => {
  // An authenticated user WITH spaces, so the card actually renders the embedded
  // SpacesListComponent (a signed-out user only shows the loading row and would
  // miss the list's DI chain — that's how the NG0201 slipped through before).
  const userState$ = new BehaviorSubject<unknown>({
    status: 'authenticated',
    user: { uid: 'u1', isAnonymous: false, emailVerified: true, providerData: [] },
    record: {
      title: 'Test User',
      spaces: { s1: { title: 'Family', type: 'family', roles: ['creator'] } },
    },
  });

  beforeEach(() =>
    TestBed.configureTestingModule({
      imports: [DebtusHomePageComponent],
      providers: [
        ...getStandardSneatProviders(debtusAppEnvironmentConfig),
        provideRouter([]),
        {
          provide: SneatUserService,
          useValue: { userState: userState$, currentUserID: 'u1' },
        },
      ],
    }),
  );

  it('renders the spaces list for a user with spaces (all DI resolves, no NG0201)', () => {
    const fixture = TestBed.createComponent(DebtusHomePageComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance).toBeTruthy();
    const host = fixture.nativeElement as HTMLElement;
    expect(host.querySelector('sneat-spaces-card')).toBeTruthy();
    expect(host.querySelector('sneat-spaces-list')).toBeTruthy();
  });
});
