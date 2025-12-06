import { ComponentFixture, TestBed } from '@angular/core/testing';

import { VolcanoPlotModal } from './volcano-plot-modal';

describe('VolcanoPlotModal', () => {
  let component: VolcanoPlotModal;
  let fixture: ComponentFixture<VolcanoPlotModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [VolcanoPlotModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(VolcanoPlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
