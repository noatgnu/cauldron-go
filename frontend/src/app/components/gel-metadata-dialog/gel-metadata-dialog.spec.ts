import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { GelMetadataDialog, GelMetadataDialogData } from './gel-metadata-dialog';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('GelMetadataDialog', () => {
  let component: GelMetadataDialog;
  let fixture: ComponentFixture<GelMetadataDialog>;
  let mockDialogRef: any;

  async function setUp(data: GelMetadataDialogData) {
    mockDialogRef = { close: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [GelMetadataDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: MAT_DIALOG_DATA, useValue: data }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(GelMetadataDialog);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }

  it('should create', async () => {
    await setUp({ metadata: {} });
    expect(component).toBeTruthy();
  });

  it('shows an empty-state message when there is no metadata', async () => {
    await setUp({ metadata: {} });
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('No embedded metadata tags were found');
  });

  it('displays metadata entries sorted alphabetically by key', async () => {
    await setUp({ metadata: { Software: 'GraphicsMagick 1.3.7', Compression: '5', DocumentName: 'D:\\scan.tif' } });

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Compression');
    expect(text).toContain('5');
    expect(text).toContain('Software');
    expect(text).toContain('GraphicsMagick 1.3.7');

    const compressionIndex = text.indexOf('Compression');
    const documentNameIndex = text.indexOf('DocumentName');
    const softwareIndex = text.indexOf('Software');
    expect(compressionIndex).toBeLessThan(documentNameIndex);
    expect(documentNameIndex).toBeLessThan(softwareIndex);
  });

  it('closes the dialog on close', async () => {
    await setUp({ metadata: {} });
    component.onClose();
    expect(mockDialogRef.close).toHaveBeenCalled();
  });
});
