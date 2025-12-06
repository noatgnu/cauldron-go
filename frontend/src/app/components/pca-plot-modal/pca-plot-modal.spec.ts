import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PcaPlotModal } from './pca-plot-modal';

describe('PcaPlotModal', () => {
  let component: PcaPlotModal;
  let fixture: ComponentFixture<PcaPlotModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PcaPlotModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PcaPlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
