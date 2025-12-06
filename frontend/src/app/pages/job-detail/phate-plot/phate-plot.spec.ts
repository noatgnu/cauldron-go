import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PhatePlot } from './phate-plot';

describe('PhatePlot', () => {
  let component: PhatePlot;
  let fixture: ComponentFixture<PhatePlot>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PhatePlot]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PhatePlot);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
