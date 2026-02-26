import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi, Mock } from 'vitest';
import { PluginInstallProgress } from './plugin-install-progress';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { Subscription } from 'rxjs';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('PluginInstallProgress', () => {
  let component: PluginInstallProgress;
  let fixture: ComponentFixture<PluginInstallProgress>;
  let wailsSpy: { listen: Mock; installPluginFromRepo: Mock; logToFile: Mock };
  let dialogRefSpy: { close: Mock };
  let notificationSpy: any;

  beforeEach(async () => {
    wailsSpy = {
      listen: vi.fn().mockReturnValue(new Subscription()),
      installPluginFromRepo: vi.fn().mockResolvedValue(undefined),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };

    dialogRefSpy = { close: vi.fn() };
    notificationSpy = {
      showSuccess: vi.fn(),
      showError: vi.fn(),
      showWarning: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [PluginInstallProgress, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsSpy },
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: NotificationService, useValue: notificationSpy },
        { provide: MAT_DIALOG_DATA, useValue: { repoURL: 'https://github.com/test/repo' } }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PluginInstallProgress);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should start installation on init', () => {
    expect(wailsSpy.installPluginFromRepo).toHaveBeenCalledWith('https://github.com/test/repo', '');
  });
});
