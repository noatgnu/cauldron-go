import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { CheckPeptideInLibraryModal } from './check-peptide-in-library-modal';
import { Wails } from '../../core/services/wails';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('CheckPeptideInLibraryModal', () => {
  let component: CheckPeptideInLibraryModal;
  let fixture: ComponentFixture<CheckPeptideInLibraryModal>;
  let dialogRefSpy: any;
  let wailsMock: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      openDataFileDialog: vi.fn(),
      parseDataFile: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [CheckPeptideInLibraryModal, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { file_path: '', peptide_col: '' } },
        { provide: Wails, useValue: wailsMock }
      ]
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
