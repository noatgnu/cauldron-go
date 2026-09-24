import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { vi } from 'vitest';
import { BatchJobRowDialog } from './batch-job-row-dialog';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

describe('BatchJobRowDialog', () => {
  let component: BatchJobRowDialog;
  let fixture: ComponentFixture<BatchJobRowDialog>;
  let mockDialogRef: any;
  let mockWails: any;
  let mockNotification: any;

  const mockPlugin = {
    id: 1,
    definition: {
      plugin: { id: 'test-plugin', name: 'Test Plugin' },
      inputs: [{ name: 'input_file', label: 'Input File', type: 'file', required: true }],
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
      imports: [BatchJobRowDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: MAT_DIALOG_DATA, useValue: { plugin: mockPlugin, initialValues: { input_file: 'a.tsv' } } },
        { provide: Wails, useValue: mockWails },
        { provide: NotificationService, useValue: mockNotification }
      ]
    })
      .compileComponents();

    fixture = TestBed.createComponent(BatchJobRowDialog);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('closes with the submitted values', () => {
    component.onFormSubmit({ input_file: 'b.tsv' });
    expect(mockDialogRef.close).toHaveBeenCalledWith({ input_file: 'b.tsv' });
  });

  it('closes with no result on cancel', () => {
    component.cancel();
    expect(mockDialogRef.close).toHaveBeenCalledWith();
  });
});
