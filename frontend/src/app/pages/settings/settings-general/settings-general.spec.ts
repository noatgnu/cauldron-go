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
      logToFile: vi.fn().mockResolvedValue(undefined)
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
});
