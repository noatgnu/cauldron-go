import { ComponentFixture, TestBed } from '@angular/core/testing';

import { EstimationPlotModal } from './estimation-plot-modal';

describe('EstimationPlotModal', () => {
  let component: EstimationPlotModal;
  let fixture: ComponentFixture<EstimationPlotModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EstimationPlotModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(EstimationPlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
