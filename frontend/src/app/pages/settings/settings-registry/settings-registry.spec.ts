import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SettingsRegistry } from './settings-registry';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('SettingsRegistry', () => {
  let component: SettingsRegistry;
  let fixture: ComponentFixture<SettingsRegistry>;
  let wailsMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    wailsMock = {
      getSettings: vi.fn().mockResolvedValue({}),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SettingsRegistry, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsRegistry);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
