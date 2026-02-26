import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { ConfirmPluginInstallDialog } from './confirm-plugin-install-dialog';
import { Wails } from '../../core/services/wails';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('ConfirmPluginInstallDialog', () => {
  let component: ConfirmPluginInstallDialog;
  let fixture: ComponentFixture<ConfirmPluginInstallDialog>;
  let dialogRefSpy: any;
  let wailsMock: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      openFile: vi.fn(),
      validateSSHKey: vi.fn(),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };

    await TestBed.configureTestingModule({
      imports: [ConfirmPluginInstallDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { repo: 'test', name: 'Test' } },
        { provide: Wails, useValue: wailsMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ConfirmPluginInstallDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
