import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PluginEnvironmentDialog } from './plugin-environment-dialog';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('PluginEnvironmentDialog', () => {
  let component: PluginEnvironmentDialog;
  let fixture: ComponentFixture<PluginEnvironmentDialog>;
  let dialogRefSpy: any;
  let wailsMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      progress$: of(null),
      getPluginEnvironmentBinding: vi.fn().mockResolvedValue(null),
      detectPythonEnvironments: vi.fn().mockResolvedValue([]),
      detectREnvironments: vi.fn().mockResolvedValue([]),
      getActivePythonEnvironment: vi.fn().mockResolvedValue(null),
      getActiveREnvironment: vi.fn().mockResolvedValue(null),
      getCustomEnvVars: vi.fn().mockResolvedValue([])
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [PluginEnvironmentDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { 
          provide: MAT_DIALOG_DATA, 
          useValue: { 
            pluginId: 'test', 
            pluginName: 'Test', 
            runtimeEnvironments: [] 
          } 
        },
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginEnvironmentDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
