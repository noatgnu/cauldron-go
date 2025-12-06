import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FuzzyClusteringPlotModal } from './fuzzy-clustering-plot-modal';

describe('FuzzyClusteringPlotModal', () => {
  let component: FuzzyClusteringPlotModal;
  let fixture: ComponentFixture<FuzzyClusteringPlotModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FuzzyClusteringPlotModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(FuzzyClusteringPlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
