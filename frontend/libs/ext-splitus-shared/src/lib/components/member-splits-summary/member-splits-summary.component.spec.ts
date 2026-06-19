import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MemberSplitsSummaryComponent } from './member-splits-summary.component';

describe('MemberSplitsSummaryComponent', () => {
  let component: MemberSplitsSummaryComponent;
  let fixture: ComponentFixture<MemberSplitsSummaryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [MemberSplitsSummaryComponent],
      schemas: [CUSTOM_ELEMENTS_SCHEMA],
    }).overrideComponent(MemberSplitsSummaryComponent, {
      set: { imports: [], schemas: [CUSTOM_ELEMENTS_SCHEMA] },
    });

    fixture = TestBed.createComponent(MemberSplitsSummaryComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
