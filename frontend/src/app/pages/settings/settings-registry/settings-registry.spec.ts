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
      setSetting: vi.fn().mockResolvedValue(undefined),
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

  it('loads both registry URLs from settings on init', async () => {
    wailsMock.getSettings = vi.fn().mockResolvedValue({
      pluginRegistryUrl: 'https://plugins.example.com',
      recipeRegistryUrl: 'https://recipes.example.com'
    });

    await component.ngOnInit();

    expect(component.pluginRegistryURL).toBe('https://plugins.example.com');
    expect(component.recipeRegistryURL).toBe('https://recipes.example.com');
  });

  it('saves the recipe registry URL', async () => {
    component.recipeRegistryURL = 'https://new-recipes.example.com';

    await component.saveRecipeRegistryURL();

    expect(wailsMock.setSetting).toHaveBeenCalledWith('recipeRegistryUrl', 'https://new-recipes.example.com');
    expect(component['config']().recipeRegistryUrl).toBe('https://new-recipes.example.com');
    expect(notificationMock.showSuccess).toHaveBeenCalled();
  });

  it('shows an error and clears saving state when saving the recipe registry URL fails', async () => {
    wailsMock.setSetting = vi.fn().mockRejectedValue(new Error('boom'));

    await component.saveRecipeRegistryURL();

    expect(notificationMock.showError).toHaveBeenCalled();
    expect(component['savingRecipeRegistry']()).toBe(false);
  });
});
