import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { BatchJobBuilder } from './batch-job-builder';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

describe('BatchJobBuilder', () => {
  let component: BatchJobBuilder;
  let fixture: ComponentFixture<BatchJobBuilder>;
  let mockWails: any;
  let mockNotification: any;
  let mockPluginService: any;
  let mockRouter: any;
  let mockDialog: any;

  const mockPlugin = {
    id: 1,
    definition: {
      plugin: { id: 'test-plugin', name: 'Test Plugin' },
      inputs: [
        { name: 'input_file', label: 'Input File', type: 'file', required: true },
        { name: 'scaler_type', label: 'Scaler', type: 'select', required: false, options: ['minmax', 'standard'] }
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

  beforeEach(async () => {
    mockWails = {
      logToFile: vi.fn().mockResolvedValue(undefined),
      openMultipleFilesDialog: vi.fn().mockResolvedValue([])
    };
    mockNotification = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };
    mockPluginService = {
      executeBatch: vi.fn().mockResolvedValue('batch-1')
    };
    mockRouter = {
      navigate: vi.fn().mockResolvedValue(true)
    };
    mockDialog = {
      open: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [BatchJobBuilder, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: mockWails },
        { provide: NotificationService, useValue: mockNotification },
        { provide: PluginV2Service, useValue: mockPluginService },
        { provide: Router, useValue: mockRouter },
        { provide: MatDialog, useValue: mockDialog }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(BatchJobBuilder);
    component = fixture.componentInstance;
    component.plugin = mockPlugin as any;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create with one empty row by default', () => {
    expect(component).toBeTruthy();
    expect(component.rows().length).toBe(1);
  });

  it('defaults the target file input to the plugin\'s first file input', () => {
    expect(component.targetFileInputName()).toBe('input_file');
  });

  it('adds an empty row', () => {
    component.addEmptyRow();
    expect(component.rows().length).toBe(2);
  });

  it('duplicates a row with a new id but the same values', () => {
    component.rows.set([{ id: 'row-1', values: { input_file: 'a.tsv' } }]);
    component.duplicateRow('row-1');
    const rows = component.rows();
    expect(rows.length).toBe(2);
    expect(rows[1].id).not.toBe('row-1');
    expect(rows[1].values).toEqual({ input_file: 'a.tsv' });
  });

  it('removes a row', () => {
    component.rows.set([{ id: 'row-1', values: {} }, { id: 'row-2', values: {} }]);
    component.removeRow('row-1');
    expect(component.rows().map(r => r.id)).toEqual(['row-2']);
  });

  it('adds one row per picked file, seeded with the target file input', async () => {
    component.rows.set([]);
    mockWails.openMultipleFilesDialog.mockResolvedValue(['a.tsv', 'b.tsv']);

    await component.addRowsFromFiles();

    const rows = component.rows();
    expect(rows.length).toBe(2);
    expect(rows.map(r => r.values['input_file'])).toEqual(['a.tsv', 'b.tsv']);
  });

  it('applies a bulk field to every row without disturbing other fields', () => {
    component.rows.set([
      { id: 'row-1', values: { input_file: 'a.tsv', scaler_type: 'minmax' } },
      { id: 'row-2', values: { input_file: 'b.tsv' } }
    ]);

    component.onBulkFieldChange('scaler_type');
    component.onBulkFormChange({ scaler_type: 'standard' });
    component.applyBulkValue();

    const rows = component.rows();
    expect(rows[0].values).toEqual({ input_file: 'a.tsv', scaler_type: 'standard' });
    expect(rows[1].values).toEqual({ input_file: 'b.tsv', scaler_type: 'standard' });
  });

  it('creates a batch from the current rows and navigates to its detail page', async () => {
    component.rows.set([
      { id: 'row-1', values: { input_file: 'a.tsv' } },
      { id: 'row-2', values: { input_file: 'b.tsv' } }
    ]);

    await component.createBatch();

    expect(mockPluginService.executeBatch).toHaveBeenCalledWith(1, 'Test Plugin', [
      { input_file: 'a.tsv' },
      { input_file: 'b.tsv' }
    ]);
    expect(mockRouter.navigate).toHaveBeenCalledWith(['/job-batch', 'batch-1']);
  });

  it('refuses to create a batch with no rows', async () => {
    component.rows.set([]);
    await component.createBatch();
    expect(mockPluginService.executeBatch).not.toHaveBeenCalled();
    expect(mockNotification.showError).toHaveBeenCalled();
  });
});
