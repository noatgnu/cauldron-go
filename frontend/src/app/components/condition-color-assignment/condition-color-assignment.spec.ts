import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ConditionColorAssignment } from './condition-color-assignment';

describe('ConditionColorAssignment', () => {
  let component: ConditionColorAssignment;
  let fixture: ComponentFixture<ConditionColorAssignment>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConditionColorAssignment]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ConditionColorAssignment);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
