import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import { SettingsGit } from './settings-git';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';

describe('SettingsGit', () => {
  let component: SettingsGit;
  let fixture: ComponentFixture<SettingsGit>;

  beforeEach(async () => {
    const wailsMock = {
      getAllGitAuthConfigs: vi.fn().mockResolvedValue([])
    };
    const notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SettingsGit],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsGit);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
