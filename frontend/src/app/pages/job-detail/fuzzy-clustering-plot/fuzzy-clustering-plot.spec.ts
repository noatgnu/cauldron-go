import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FuzzyClusteringPlot } from './fuzzy-clustering-plot';

describe('FuzzyClusteringPlot', () => {
  let component: FuzzyClusteringPlot;
  let fixture: ComponentFixture<FuzzyClusteringPlot>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FuzzyClusteringPlot]
    })
    .compileComponents();

    fixture = TestBed.createComponent(FuzzyClusteringPlot);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
