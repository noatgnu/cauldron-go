import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MsfraggerToCurtainptmModal } from './msfragger-to-curtainptm-modal';
import { Wails } from '../../core/services/wails';
import { vi } from 'vitest';

describe('MsfraggerToCurtainptmModal', () => {
  let component: MsfraggerToCurtainptmModal;
  let fixture: ComponentFixture<MsfraggerToCurtainptmModal>;
  let dialogRefSpy: any;
  let wailsMock: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      openDataFileDialog: vi.fn(),
      parseDataFile: vi.fn(),
      getImportedFiles: vi.fn().mockResolvedValue([])
    };

    await TestBed.configureTestingModule({
      imports: [
        MsfraggerToCurtainptmModal,
        ReactiveFormsModule,
        NoopAnimationsModule
      ],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: {} },
        { provide: Wails, useValue: wailsMock }
      ]
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
