import { ComponentFixture, TestBed } from '@angular/core/testing';

import { MsfraggerToCurtainptmModal } from './msfragger-to-curtainptm-modal';

describe('MsfraggerToCurtainptmModal', () => {
  let component: MsfraggerToCurtainptmModal;
  let fixture: ComponentFixture<MsfraggerToCurtainptmModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MsfraggerToCurtainptmModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(MsfraggerToCurtainptmModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
