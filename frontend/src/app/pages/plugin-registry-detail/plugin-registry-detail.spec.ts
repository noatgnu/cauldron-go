import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { PluginRegistryDetail } from './plugin-registry-detail';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { DomSanitizer } from '@angular/platform-browser';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('PluginRegistryDetail', () => {
  let component: PluginRegistryDetail;
  let fixture: ComponentFixture<PluginRegistryDetail>;
  let activatedRouteMock: any;
  let routerMock: any;
  let wailsMock: any;
  let notificationMock: any;
  let pluginV2ServiceMock: any;
  let sanitizerMock: any;

  beforeEach(async () => {
    activatedRouteMock = {
      paramMap: of({ get: () => '1' }),
      snapshot: {
        paramMap: {
          get: vi.fn().mockReturnValue('1')
        }
      }
    };
    routerMock = {
      navigate: vi.fn()
    };
    wailsMock = {
      getRegistryPlugin: vi.fn().mockResolvedValue({ id: '1', name: 'Test', description: 'Test', author: { name: 'Test' }, categories: [] }),
      isPluginInstalled: vi.fn().mockResolvedValue(false),
      getPluginVersion: vi.fn().mockResolvedValue('1.0.0'),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };
    pluginV2ServiceMock = {
      getAllPlugins: vi.fn().mockResolvedValue([])
    };
    sanitizerMock = {
      bypassSecurityTrustHtml: vi.fn().mockImplementation((val) => val)
    };

    await TestBed.configureTestingModule({
      imports: [PluginRegistryDetail, NoopAnimationsModule],
      providers: [
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: Router, useValue: routerMock },
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: PluginV2Service, useValue: pluginV2ServiceMock },
        { provide: DomSanitizer, useValue: sanitizerMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginRegistryDetail);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
