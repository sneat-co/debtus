import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { getStandardSneatProviders } from '@sneat/app';
import { debtusAppEnvironmentConfig } from '../../environments/environment';
import { DebtusHomePageComponent } from './debtus-home-page.component';

// Uses the app's real provider set (getStandardSneatProviders) so the full DI
// chain of the embedded SpacesCardComponent is exercised. This is the test that
// catches a missing provider (e.g. SpaceService) which only surfaces at runtime
// as NG0201, not at build time.
describe('DebtusHomePageComponent', () => {
  beforeEach(() =>
    TestBed.configureTestingModule({
      imports: [DebtusHomePageComponent],
      providers: [
        ...getStandardSneatProviders(debtusAppEnvironmentConfig),
        provideRouter([]),
      ],
    }),
  );

  it('creates and renders the shared spaces card (all DI resolves)', () => {
    const fixture = TestBed.createComponent(DebtusHomePageComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance).toBeTruthy();
    const host = fixture.nativeElement as HTMLElement;
    expect(host.querySelector('sneat-spaces-card')).toBeTruthy();
  });
});
