import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DiannCvModal } from './diann-cv-modal';

describe('DiannCvModal', () => {
  let component: DiannCvModal;
  let fixture: ComponentFixture<DiannCvModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DiannCvModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(DiannCvModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
