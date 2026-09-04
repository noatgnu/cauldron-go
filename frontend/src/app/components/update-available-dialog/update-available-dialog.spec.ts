import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { vi } from 'vitest';
import { UpdateAvailableDialog } from './update-available-dialog';
import { UpdateInfo } from '../../core/services/wails';

const sampleInfo: UpdateInfo = {
  currentVersion: 'v0.0.9',
  latestVersion: 'v0.1.0',
  available: true,
  releaseUrl: 'https://github.com/noatgnu/cauldron-go/releases/tag/v0.1.0',
  releaseNotes: 'New stuff',
  publishedAt: '2026-09-04T00:00:00Z'
};

describe('UpdateAvailableDialog', () => {
  let component: UpdateAvailableDialog;
  let fixture: ComponentFixture<UpdateAvailableDialog>;
  let dialogRefSpy: any;
  let openSpy: any;

  beforeEach(async () => {
    dialogRefSpy = { close: vi.fn() };
    openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    await TestBed.configureTestingModule({
      imports: [UpdateAvailableDialog],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: sampleInfo }
      ]
    })
      .compileComponents();

    fixture = TestBed.createComponent(UpdateAvailableDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  afterEach(() => {
    openSpy.mockRestore();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('exposes the injected update info', () => {
    expect(component.data).toEqual(sampleInfo);
  });

  it('opens the release URL in a new tab on viewRelease', () => {
    component.viewRelease();
    expect(openSpy).toHaveBeenCalledWith(sampleInfo.releaseUrl, '_blank');
  });

  it('does nothing on viewRelease when no release URL is present', () => {
    (component.data as any).releaseUrl = '';
    component.viewRelease();
    expect(openSpy).not.toHaveBeenCalled();
  });

  it('closes the dialog on close', () => {
    component.close();
    expect(dialogRefSpy.close).toHaveBeenCalled();
  });

  it('renders the version comparison and release notes', () => {
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('v0.1.0');
    expect(text).toContain('v0.0.9');
    expect(text).toContain('New stuff');
  });
});
