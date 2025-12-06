import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ViolinPlot } from './violin-plot';

describe('ViolinPlot', () => {
  let component: ViolinPlot;
  let fixture: ComponentFixture<ViolinPlot>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ViolinPlot]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ViolinPlot);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
