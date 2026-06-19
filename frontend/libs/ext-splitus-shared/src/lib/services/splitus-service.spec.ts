import { TestBed } from '@angular/core/testing';
import { SneatApiService } from '@sneat/api';
import { SplitusService } from './splitus-service';

describe('SplitusService', () => {
  beforeEach(() =>
    TestBed.configureTestingModule({
      providers: [
        SplitusService,
        { provide: SneatApiService, useValue: { post: vi.fn() } },
      ],
    }),
  );

  it('should be created', () => {
    expect(TestBed.inject(SplitusService)).toBeTruthy();
  });
});
