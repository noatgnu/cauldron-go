import { ComponentFixture, TestBed } from '@angular/core/testing';

import { EstimationPlot } from './estimation-plot';

describe('EstimationPlot', () => {
  let component: EstimationPlot;
  let fixture: ComponentFixture<EstimationPlot>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EstimationPlot]
    })
    .compileComponents();

    fixture = TestBed.createComponent(EstimationPlot);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
