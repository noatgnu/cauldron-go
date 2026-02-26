import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { UniprotModal } from './uniprot-modal';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('UniprotModal', () => {
  let component: UniprotModal;
  let fixture: ComponentFixture<UniprotModal>;
  let dialogRefSpy: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [UniprotModal, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { columns: [] } }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(UniprotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
