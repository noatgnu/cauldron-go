import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { DiannToCurtainptmModal } from './diann-to-curtainptm-modal';
import { Wails } from '../../core/services/wails';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { ReactiveFormsModule } from '@angular/forms';

describe('DiannToCurtainptmModal', () => {
  let component: DiannToCurtainptmModal;
  let fixture: ComponentFixture<DiannToCurtainptmModal>;
  let dialogRefSpy: any;
  let wailsMock: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      openDataFileDialog: vi.fn(),
      parseDataFile: vi.fn(),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };

    await TestBed.configureTestingModule({
      imports: [
        DiannToCurtainptmModal, 
        NoopAnimationsModule,
        ReactiveFormsModule
      ],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: {} },
        { provide: Wails, useValue: wailsMock }
      ]
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
