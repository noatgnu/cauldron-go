import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PhatePlotModal } from './phate-plot-modal';

describe('PhatePlotModal', () => {
  let component: PhatePlotModal;
  let fixture: ComponentFixture<PhatePlotModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PhatePlotModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PhatePlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
