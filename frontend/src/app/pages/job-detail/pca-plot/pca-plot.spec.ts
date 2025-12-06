import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PcaPlot } from './pca-plot';

describe('PcaPlot', () => {
  let component: PcaPlot;
  let fixture: ComponentFixture<PcaPlot>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PcaPlot]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PcaPlot);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
