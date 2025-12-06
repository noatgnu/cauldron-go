import { ComponentFixture, TestBed } from '@angular/core/testing';

import { CorrelationMatrixModal } from './correlation-matrix-modal';

describe('CorrelationMatrixModal', () => {
  let component: CorrelationMatrixModal;
  let fixture: ComponentFixture<CorrelationMatrixModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CorrelationMatrixModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CorrelationMatrixModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
