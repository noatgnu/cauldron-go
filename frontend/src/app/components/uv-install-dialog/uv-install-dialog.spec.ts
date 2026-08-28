import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogRef } from '@angular/material/dialog';
import { signal } from '@angular/core';
import { vi } from 'vitest';
import { UvInstallDialog } from './uv-install-dialog';
import { Wails } from '../../core/services/wails';

describe('UvInstallDialog', () => {
  let component: UvInstallDialog;
  let fixture: ComponentFixture<UvInstallDialog>;
  let wailsMock: {
    progress: ReturnType<typeof signal>;
    isUvAvailable: ReturnType<typeof vi.fn>;
    getUvPath: ReturnType<typeof vi.fn>;
    downloadUv: ReturnType<typeof vi.fn>;
    installUvPythonVersion: ReturnType<typeof vi.fn>;
    logToFile: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    wailsMock = {
      progress: signal(null),
      isUvAvailable: vi.fn().mockResolvedValue(false),
      getUvPath: vi.fn().mockResolvedValue(''),
      downloadUv: vi.fn().mockResolvedValue(undefined),
      installUvPythonVersion: vi.fn().mockResolvedValue(undefined),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };

    await TestBed.configureTestingModule({
      imports: [UvInstallDialog],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: MatDialogRef, useValue: { close: () => {} } }
      ]
    })
      .compileComponents();

    fixture = TestBed.createComponent(UvInstallDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should check uv availability on init', () => {
    expect(wailsMock.isUvAvailable).toHaveBeenCalled();
  });
});
