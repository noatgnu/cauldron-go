import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi, Mock } from 'vitest';
import { InstallPluginDialog } from './install-plugin-dialog';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

describe('InstallPluginDialog', () => {
  let component: InstallPluginDialog;
  let fixture: ComponentFixture<InstallPluginDialog>;
  let dialogRefSpy: { close: Mock };
  let wailsMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    dialogRefSpy = { close: vi.fn() };
    wailsMock = {
      openFile: vi.fn(),
      validateSSHKey: vi.fn()
    };
    notificationMock = {
      showError: vi.fn(),
      showWarning: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [InstallPluginDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: {} },
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(InstallPluginDialog);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with empty form', () => {
    expect(component.form.get('repoURL')?.value).toBe('');
  });

  it('should validate repository URL pattern', () => {
    const repoURLControl = component.form.get('repoURL');

    repoURLControl?.setValue('');
    expect(repoURLControl?.hasError('required')).toBe(true);

    repoURLControl?.setValue('invalid-url');
    expect(repoURLControl?.hasError('pattern')).toBe(true);

    repoURLControl?.setValue('https://github.com/user/repo');
    expect(repoURLControl?.valid).toBe(true);

    repoURLControl?.setValue('https://github.com/user/repo.git');
    expect(repoURLControl?.valid).toBe(true);

    repoURLControl?.setValue('git@github.com:user/repo.git');
    expect(repoURLControl?.valid).toBe(true);
  });

  it('should close dialog with repo URL on install', () => {
    const testURL = 'https://github.com/user/repo';
    component.form.patchValue({ repoURL: testURL });

    component.install();

    expect(dialogRefSpy.close).toHaveBeenCalledWith({
      repoURL: testURL,
      commitHash: '',
      sshKeyPath: undefined,
      passphrase: undefined
    });
  });

  it('should not close dialog if form is invalid', () => {
    component.form.get('repoURL')?.setValue('');

    component.install();

    expect(dialogRefSpy.close).not.toHaveBeenCalled();
  });

  it('should close dialog without data on cancel', () => {
    component.cancel();

    expect(dialogRefSpy.close).toHaveBeenCalledWith();
  });

  it('should display correct error messages', () => {
    const repoURLControl = component.form.get('repoURL');

    repoURLControl?.setValue('');
    repoURLControl?.markAsTouched();
    expect(component.getErrorMessage()).toBe('Repository URL is required');

    repoURLControl?.setValue('invalid');
    expect(component.getErrorMessage()).toBe('Invalid repository URL format. Expected: https://github.com/user/repo');

    repoURLControl?.setValue('https://github.com/user/repo');
    expect(component.getErrorMessage()).toBe('');
  });
});
