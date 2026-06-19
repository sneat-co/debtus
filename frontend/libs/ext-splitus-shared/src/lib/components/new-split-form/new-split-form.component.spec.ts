import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ErrorLogger } from '@sneat/core';
import { SplitusService } from '../../services/splitus-service';
import { NewSplitFormComponent } from './new-split-form.component';

describe('NewSplitFormComponent', () => {
  let component: NewSplitFormComponent;
  let fixture: ComponentFixture<NewSplitFormComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NewSplitFormComponent],
      schemas: [CUSTOM_ELEMENTS_SCHEMA],
      providers: [
        {
          provide: ErrorLogger,
          useValue: { logError: vi.fn(), logErrorHandler: () => vi.fn() },
        },
        {
          provide: SplitusService,
          useValue: { createSplitRecord: vi.fn() },
        },
      ],
    }).overrideComponent(NewSplitFormComponent, {
      set: {
        imports: [],
        providers: [],
        schemas: [CUSTOM_ELEMENTS_SCHEMA],
      },
    });

    fixture = TestBed.createComponent(NewSplitFormComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
