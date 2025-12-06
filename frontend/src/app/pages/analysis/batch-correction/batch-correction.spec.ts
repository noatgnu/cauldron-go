import { ComponentFixture, TestBed } from '@angular/core/testing';

import { BatchCorrection } from './batch-correction';

describe('BatchCorrection', () => {
  let component: BatchCorrection;
  let fixture: ComponentFixture<BatchCorrection>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [BatchCorrection]
    })
    .compileComponents();

    fixture = TestBed.createComponent(BatchCorrection);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
