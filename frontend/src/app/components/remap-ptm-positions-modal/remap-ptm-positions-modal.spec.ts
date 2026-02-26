import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { RemapPtmPositionsModal } from './remap-ptm-positions-modal';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('RemapPtmPositionsModal', () => {
  let component: RemapPtmPositionsModal;
  let fixture: ComponentFixture<RemapPtmPositionsModal>;
  let dialogRefSpy: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [RemapPtmPositionsModal, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { columns: [] } }
      ]
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
