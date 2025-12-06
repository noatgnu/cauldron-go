import { ComponentFixture, TestBed } from '@angular/core/testing';

import { CvPlot } from './cv-plot';

describe('CvPlot', () => {
  let component: CvPlot;
  let fixture: ComponentFixture<CvPlot>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CvPlot]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CvPlot);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
