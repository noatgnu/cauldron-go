import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SettingsGeneral } from './settings-general';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('SettingsGeneral', () => {
  let component: SettingsGeneral;
  let fixture: ComponentFixture<SettingsGeneral>;
  let wailsMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    wailsMock = {
      getSettings: vi.fn().mockResolvedValue({}),
      getRemotePlugins: vi.fn().mockResolvedValue([]),
      logToFile: vi.fn().mockResolvedValue(undefined),
      setSetting: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SettingsGeneral, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsGeneral);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('defaults debug mode to off when not set', async () => {
    await component.loadSettings();
    expect(component['debugMode']()).toBe(false);
  });

  it('loads debug mode from settings', async () => {
    wailsMock.getSettings.mockResolvedValue({ debugMode: true });
    await component.loadSettings();
    expect(component['debugMode']()).toBe(true);
  });

  it('persists debug mode changes', async () => {
    await component.setDebugMode(true);
    expect(component['debugMode']()).toBe(true);
    expect(wailsMock.setSetting).toHaveBeenCalledWith('debugMode', true);
  });
});
