import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { ImportDialog } from './import-dialog';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('ImportDialog', () => {
  let component: ImportDialog;
  let fixture: ComponentFixture<ImportDialog>;
  let dialogRefSpy: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [ImportDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: {} }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ImportDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
