import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { ProtocolIoCitationExportModal } from './protocol-io-citation-export-modal';
import { vi } from 'vitest';

describe('ProtocolIoCitationExportModal', () => {
  let component: ProtocolIoCitationExportModal;
  let fixture: ComponentFixture<ProtocolIoCitationExportModal>;
  let dialogRefSpy: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [ProtocolIoCitationExportModal],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: {} }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ProtocolIoCitationExportModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
