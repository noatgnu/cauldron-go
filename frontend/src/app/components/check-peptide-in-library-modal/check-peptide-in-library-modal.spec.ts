import { ComponentFixture, TestBed } from '@angular/core/testing';

import { CheckPeptideInLibraryModal } from './check-peptide-in-library-modal';

describe('CheckPeptideInLibraryModal', () => {
  let component: CheckPeptideInLibraryModal;
  let fixture: ComponentFixture<CheckPeptideInLibraryModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CheckPeptideInLibraryModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CheckPeptideInLibraryModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
