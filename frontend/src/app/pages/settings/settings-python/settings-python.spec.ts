import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SettingsPython } from './settings-python';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('SettingsPython', () => {
  let component: SettingsPython;
  let fixture: ComponentFixture<SettingsPython>;
  let wailsMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    wailsMock = {
      getSettings: vi.fn().mockResolvedValue({}),
      detectPythonEnvironments: vi.fn().mockResolvedValue([]),
      getActivePythonEnvironment: vi.fn().mockResolvedValue(null),
      getPythonVersion: vi.fn().mockResolvedValue('3.10.0'),
      checkDockerVersion: vi.fn().mockResolvedValue('20.10.0'),
      getVirtualEnvironments: vi.fn().mockResolvedValue([]),
      bindingsUpdated$: of(undefined),
      progress$: of(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SettingsPython, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsPython);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
