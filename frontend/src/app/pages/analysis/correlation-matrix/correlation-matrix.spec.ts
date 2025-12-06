import { ComponentFixture, TestBed } from '@angular/core/testing';

import { CorrelationMatrix } from './correlation-matrix';

describe('CorrelationMatrix', () => {
  let component: CorrelationMatrix;
  let fixture: ComponentFixture<CorrelationMatrix>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CorrelationMatrix]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CorrelationMatrix);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
