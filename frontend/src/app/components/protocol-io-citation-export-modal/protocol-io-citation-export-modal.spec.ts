import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ProtocolIoCitationExportModal } from './protocol-io-citation-export-modal';

describe('ProtocolIoCitationExportModal', () => {
  let component: ProtocolIoCitationExportModal;
  let fixture: ComponentFixture<ProtocolIoCitationExportModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProtocolIoCitationExportModal]
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
