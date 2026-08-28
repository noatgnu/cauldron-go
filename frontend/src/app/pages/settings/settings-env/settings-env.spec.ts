import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import { SettingsEnv } from './settings-env';
import { Wails } from '../../../core/services/wails';
import { PluginV2Service } from '../../../core/services/plugin-v2';
import { NotificationService } from '../../../core/services/notification.service';

describe('SettingsEnv', () => {
  let component: SettingsEnv;
  let fixture: ComponentFixture<SettingsEnv>;

  beforeEach(async () => {
    const wailsMock = {
      getGlobalCustomEnvVars: vi.fn().mockResolvedValue([]),
      getCustomEnvVars: vi.fn().mockResolvedValue([])
    };
    const pluginServiceMock = {
      getAllPlugins: vi.fn().mockResolvedValue([])
    };
    const notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SettingsEnv],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: PluginV2Service, useValue: pluginServiceMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsEnv);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
