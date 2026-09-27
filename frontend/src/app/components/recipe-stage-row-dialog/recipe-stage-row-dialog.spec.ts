import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { vi } from 'vitest';
import { RecipeStageRowDialog } from './recipe-stage-row-dialog';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

describe('RecipeStageRowDialog', () => {
  let component: RecipeStageRowDialog;
  let fixture: ComponentFixture<RecipeStageRowDialog>;
  let mockDialogRef: any;
  let mockWails: any;
  let mockNotification: any;

  const mockPlugin = {
    id: 2,
    definition: {
      plugin: { id: 'downstream-plugin', name: 'Downstream Plugin' },
      inputs: [
        { name: 'input_file', label: 'Input File', type: 'file', required: true },
        { name: 'threshold', label: 'Threshold', type: 'number', required: false, default: 1 }
      ],
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

  const earlierStages = [
    { index: 0, label: 'Stage 1: Upstream Plugin', outputs: [{ name: 'result' }] }
  ];

  beforeEach(async () => {
    mockDialogRef = {
      close: vi.fn()
    };
    mockWails = {
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    mockNotification = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [RecipeStageRowDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        {
          provide: MAT_DIALOG_DATA,
          useValue: { plugin: mockPlugin, initialValues: {}, initialBindings: {}, earlierStages }
        },
        { provide: Wails, useValue: mockWails },
        { provide: NotificationService, useValue: mockNotification }
      ]
    })
      .compileComponents();

    fixture = TestBed.createComponent(RecipeStageRowDialog);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('starts with the file input unbound and included in the form plugin', () => {
    expect(component.isBound('input_file')).toBe(false);
    expect(component.formPlugin().definition.inputs.map((i: any) => i.name)).toContain('input_file');
  });

  it('binding an input removes it from the form plugin and defaults to the first earlier output', () => {
    component.setMode('input_file', 'bound');
    expect(component.isBound('input_file')).toBe(true);
    expect(component.bindingFor('input_file')).toEqual({ stageIndex: 0, outputName: 'result' });
    expect(component.formPlugin().definition.inputs.map((i: any) => i.name)).not.toContain('input_file');
    expect(component.formPlugin().definition.inputs.map((i: any) => i.name)).toContain('threshold');
  });

  it('switching back to file mode removes the binding and restores the field to the form plugin', () => {
    component.setMode('input_file', 'bound');
    component.setMode('input_file', 'file');
    expect(component.isBound('input_file')).toBe(false);
    expect(component.formPlugin().definition.inputs.map((i: any) => i.name)).toContain('input_file');
  });

  it('closes with the submitted values and current bindings', () => {
    component.setMode('input_file', 'bound');
    component.onFormSubmit({ threshold: 42 });
    expect(mockDialogRef.close).toHaveBeenCalledWith({
      values: { threshold: 42 },
      bindings: { input_file: { stageIndex: 0, outputName: 'result' } }
    });
  });

  it('closes with no result on cancel', () => {
    component.cancel();
    expect(mockDialogRef.close).toHaveBeenCalledWith();
  });
});
