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

  describe('column-selector with a bound (filtered-out) sourceFile', () => {
    const boundPlugin = {
      ...mockPlugin,
      definition: {
        ...mockPlugin.definition,
        inputs: [
          { name: 'id_vars', label: 'ID Columns', type: 'column-selector', multiple: true, sourceFile: 'input_file', required: true },
          { name: 'names_from', label: 'Names From Column', type: 'column-selector', multiple: false, sourceFile: 'input_file', required: true }
        ]
      }
    };

    beforeEach(async () => {
      component.plugin = boundPlugin as any;
      await (component as any).initializeForm();
      fixture.detectChanges();
    });

    it('reports no source file input present', () => {
      expect(component.hasSourceFileInput(boundPlugin.definition.inputs[0] as any)).toBe(false);
    });

    it('parses comma-separated free text into an array for a multiple column-selector', () => {
      component.onFreeTextColumnsInput('id_vars', 'Protein.Group, Genes');
      expect(component.form.get('id_vars')?.value).toEqual(['Protein.Group', 'Genes']);
    });

    it('trims and drops empty entries from free text input', () => {
      component.onFreeTextColumnsInput('id_vars', ' Protein.Group ,, Genes ,');
      expect(component.form.get('id_vars')?.value).toEqual(['Protein.Group', 'Genes']);
    });

    it('preserves raw in-progress text for display without collapsing a trailing comma', () => {
      component.onFreeTextColumnsInput('id_vars', 'Protein.Group,');
      expect(component.getFreeTextColumnsValue('id_vars')).toBe('Protein.Group,');
      expect(component.form.get('id_vars')?.value).toEqual(['Protein.Group']);
    });

    it('accepts a plain string via formControlName for a non-multiple bound column-selector', () => {
      component.form.get('names_from')?.setValue('Sample');
      expect(component.form.get('names_from')?.value).toBe('Sample');
    });
  });

  describe('column-selector with its sourceFile input present', () => {
    const unboundPlugin = {
      ...mockPlugin,
      definition: {
        ...mockPlugin.definition,
        inputs: [
          { name: 'input_file', label: 'Input File', type: 'file', required: true },
          { name: 'id_vars', label: 'ID Columns', type: 'column-selector', multiple: true, sourceFile: 'input_file', required: true }
        ]
      }
    };

    beforeEach(async () => {
      component.plugin = unboundPlugin as any;
      await (component as any).initializeForm();
      fixture.detectChanges();
    });

    it('reports the source file input as present', () => {
      expect(component.hasSourceFileInput(unboundPlugin.definition.inputs[1] as any)).toBe(true);
    });
  });
});
