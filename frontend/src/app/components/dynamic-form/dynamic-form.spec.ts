import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { DynamicFormComponent } from './dynamic-form';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('DynamicFormComponent', () => {
  let component: DynamicFormComponent;
  let fixture: ComponentFixture<DynamicFormComponent>;
  let mockWails: any;
  let mockNotificationService: any;

  const mockPlugin = {
    id: 1,
    definition: {
      plugin: {
        id: 'test-plugin',
        name: 'Test Plugin'
      },
      inputs: [],
      runtime: { environments: ['python'], entrypoint: 'script' },
      execution: {}
    },
    folderPath: '/test/path',
    scriptPath: '/test/path/script.py',
    installSource: 'builtin',
    commitHash: '',
    repository: '',
    enabled: true
  };

  beforeEach(async () => {
    mockWails = {
      logToFile: vi.fn().mockResolvedValue(undefined),
      readFile: vi.fn().mockResolvedValue(''),
      getPluginExampleFilePath: vi.fn().mockResolvedValue('')
    };

    mockNotificationService = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [DynamicFormComponent, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: mockWails },
        { provide: NotificationService, useValue: mockNotificationService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(DynamicFormComponent);
    component = fixture.componentInstance;
    component.plugin = mockPlugin as any;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should build empty form when no inputs', () => {
    expect(component.form).toBeDefined();
  });
});
