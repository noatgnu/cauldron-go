import { ComponentFixture, TestBed } from '@angular/core/testing';

import { RemapPtmPositionsModal } from './remap-ptm-positions-modal';

describe('RemapPtmPositionsModal', () => {
  let component: RemapPtmPositionsModal;
  let fixture: ComponentFixture<RemapPtmPositionsModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RemapPtmPositionsModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(RemapPtmPositionsModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
