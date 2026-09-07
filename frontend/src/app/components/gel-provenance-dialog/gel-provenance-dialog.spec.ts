import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { GelProvenanceDialog, GelProvenanceDialogData } from './gel-provenance-dialog';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { GelAnalysisProvenance } from '../../core/services/wails';

describe('GelProvenanceDialog', () => {
  let component: GelProvenanceDialog;
  let fixture: ComponentFixture<GelProvenanceDialog>;
  let mockDialogRef: any;
  let mockData: GelProvenanceDialogData;

  const sampleProvenance: GelAnalysisProvenance = {
    generatedAt: '2026-09-05T00:00:00Z',
    appVersion: 'v0.0.11-dirty',
    analysisEngineVersion: '1.0.0',
    imagePath: '/tmp/sample.png',
    imageSha256: 'abc123',
    lanes: [{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false }],
    autoDetectUsed: false
  } as GelAnalysisProvenance;

  beforeEach(async () => {
    mockDialogRef = { close: vi.fn() };
    mockData = { provenance: sampleProvenance };

    await TestBed.configureTestingModule({
      imports: [GelProvenanceDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: MAT_DIALOG_DATA, useValue: mockData }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(GelProvenanceDialog);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('displays the image hash and engine version', () => {
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('abc123');
    expect(text).toContain('1.0.0');
  });

  it('hides Python-specific fields when auto-detect was not used', () => {
    const text = fixture.nativeElement.textContent as string;
    expect(text).not.toContain('Python version');
  });

  it('shows Python environment details when auto-detect was used', async () => {
    const autoDetectData: GelProvenanceDialogData = {
      provenance: {
        ...sampleProvenance,
        autoDetectUsed: true,
        pythonVersion: 'Python 3.11.4',
        pythonPackages: ['numpy==1.26.0', 'scipy==1.11.0'],
        autoDetectScriptSha256: 'def456'
      } as GelAnalysisProvenance
    };

    TestBed.resetTestingModule();
    await TestBed.configureTestingModule({
      imports: [GelProvenanceDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: MAT_DIALOG_DATA, useValue: autoDetectData }
      ]
    }).compileComponents();

    const autoFixture = TestBed.createComponent(GelProvenanceDialog);
    autoFixture.detectChanges();

    const text = autoFixture.nativeElement.textContent as string;
    expect(text).toContain('Python 3.11.4');
    expect(text).toContain('numpy==1.26.0');
    expect(text).toContain('def456');
  });

  it('closes the dialog on close', () => {
    component.onClose();
    expect(mockDialogRef.close).toHaveBeenCalled();
  });
});
