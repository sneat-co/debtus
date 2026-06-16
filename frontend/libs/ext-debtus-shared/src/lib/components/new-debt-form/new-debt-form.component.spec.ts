import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ErrorLogger } from '@sneat/core';
import { DebtusService } from '../../services/debtus-service';
import { NewDebtFormComponent } from './new-debt-form.component';

describe('NewDebtFormComponent', () => {
  let component: NewDebtFormComponent;
  let fixture: ComponentFixture<NewDebtFormComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NewDebtFormComponent],
      schemas: [CUSTOM_ELEMENTS_SCHEMA],
      providers: [
        {
          provide: ErrorLogger,
          useValue: { logError: vi.fn(), logErrorHandler: () => vi.fn() },
        },
        {
          provide: DebtusService,
          useValue: { createDebtRecord: vi.fn() },
        },
      ],
    }).overrideComponent(NewDebtFormComponent, {
      set: {
        imports: [],
        providers: [],
        schemas: [CUSTOM_ELEMENTS_SCHEMA],
      },
    });

    fixture = TestBed.createComponent(NewDebtFormComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
