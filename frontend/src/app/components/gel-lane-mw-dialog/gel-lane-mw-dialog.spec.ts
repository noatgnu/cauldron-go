import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { GelLaneMwDialog, GelLaneMwDialogData } from './gel-lane-mw-dialog';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('GelLaneMwDialog', () => {
  let component: GelLaneMwDialog;
  let fixture: ComponentFixture<GelLaneMwDialog>;
  let mockDialogRef: any;
  let mockData: GelLaneMwDialogData;

  beforeEach(async () => {
    mockDialogRef = { close: vi.fn() };
    mockData = { laneLabel: 'Lane 1' };

    await TestBed.configureTestingModule({
      imports: [GelLaneMwDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: MAT_DIALOG_DATA, useValue: mockData }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(GelLaneMwDialog);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('pre-fills the input from existing markerMWs', async () => {
    mockData.markerMWs = [250, 100, 25];
    fixture = TestBed.createComponent(GelLaneMwDialog);
    component = fixture.componentInstance;
    fixture.detectChanges();
    expect(component.inputValue()).toBe('250, 100, 25');
  });

  it('parses a comma-separated list and closes with the parsed numbers', () => {
    component.onInputChange('250, 150, 100, 75, 50, 37, 25');
    component.onConfirm();
    expect(mockDialogRef.close).toHaveBeenCalledWith([250, 150, 100, 75, 50, 37, 25]);
  });

  it('rejects an empty input', () => {
    component.onInputChange('   ');
    component.onConfirm();
    expect(mockDialogRef.close).not.toHaveBeenCalled();
    expect(component.error()).toContain('at least one');
  });

  it('rejects a non-numeric entry', () => {
    component.onInputChange('250, abc, 25');
    component.onConfirm();
    expect(mockDialogRef.close).not.toHaveBeenCalled();
    expect(component.error()).toContain('abc');
  });

  it('closes with null on cancel', () => {
    component.onCancel();
    expect(mockDialogRef.close).toHaveBeenCalledWith(null);
  });
});
