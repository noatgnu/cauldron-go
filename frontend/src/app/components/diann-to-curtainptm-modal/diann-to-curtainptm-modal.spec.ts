import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DiannToCurtainptmModal } from './diann-to-curtainptm-modal';

describe('DiannToCurtainptmModal', () => {
  let component: DiannToCurtainptmModal;
  let fixture: ComponentFixture<DiannToCurtainptmModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DiannToCurtainptmModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(DiannToCurtainptmModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
