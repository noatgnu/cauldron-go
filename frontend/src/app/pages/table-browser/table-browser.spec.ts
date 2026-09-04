import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { vi } from 'vitest';
import { TableBrowser } from './table-browser';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

const sampleInfo = {
  path: '/data/sample.parquet',
  columns: [
    { name: 'id', type: 'INT64' },
    { name: 'name', type: 'BYTE_ARRAY' }
  ],
  numRows: 100,
  numRowGroups: 2,
  fileSize: 4096
};

const sampleCSVInfo = {
  path: '/data/sample.csv',
  columns: [
    { name: 'id', type: '' },
    { name: 'name', type: '' }
  ],
  numRows: 100,
  numRowGroups: 0,
  fileSize: 4096
};

describe('TableBrowser', () => {
  let component: TableBrowser;
  let fixture: ComponentFixture<TableBrowser>;
  let wailsMock: any;
  let notificationMock: any;

  function createComponent() {
    fixture = TestBed.createComponent(TableBrowser);
    component = fixture.componentInstance;
  }

  function state(): any {
    return component as any;
  }

  beforeEach(async () => {
    wailsMock = {
      progress: signal(null),
      openTableFileDialog: vi.fn().mockResolvedValue('/data/sample.parquet'),
      getTableFileInfo: vi.fn().mockResolvedValue(sampleInfo),
      getTableFilePage: vi.fn().mockResolvedValue([{ id: 1, name: 'a' }, { id: 2, name: 'b' }]),
      saveTableExportDialog: vi.fn().mockResolvedValue('/data/sample.csv'),
      exportTableFile: vi.fn().mockResolvedValue(undefined),
      closeTableFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn(),
      showWarning: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [TableBrowser],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
      .compileComponents();
  });

  it('should create', async () => {
    createComponent();
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('loads file info and the first page when a file is chosen', async () => {
    createComponent();
    await fixture.whenStable();

    await component.openFile();

    expect(wailsMock.getTableFileInfo).toHaveBeenCalledWith('/data/sample.parquet');
    expect(wailsMock.getTableFilePage).toHaveBeenCalledWith('/data/sample.parquet', 0, 25);
    expect(state().fileInfo()).toEqual(sampleInfo);
    expect(state().rows().length).toBe(2);
  });

  it('does nothing when the file dialog is cancelled', async () => {
    wailsMock.openTableFileDialog.mockResolvedValue('');
    createComponent();
    await fixture.whenStable();

    await component.openFile();

    expect(wailsMock.getTableFileInfo).not.toHaveBeenCalled();
    expect(state().fileInfo()).toBeNull();
  });

  it('defaults selected columns to all columns on open', async () => {
    createComponent();
    await fixture.whenStable();

    await component.openFile();

    expect(component.isColumnSelected('id')).toBe(true);
    expect(component.isColumnSelected('name')).toBe(true);
  });

  it('requests the correct offset when the page changes', async () => {
    createComponent();
    await fixture.whenStable();
    await component.openFile();

    await component.onPageChange({ pageIndex: 2, pageSize: 25 } as any);

    expect(wailsMock.getTableFilePage).toHaveBeenCalledWith('/data/sample.parquet', 50, 25);
  });

  it('toggles column selection', async () => {
    createComponent();
    await fixture.whenStable();
    await component.openFile();

    component.toggleColumn('name');
    expect(component.isColumnSelected('name')).toBe(false);

    component.toggleColumn('name');
    expect(component.isColumnSelected('name')).toBe(true);
  });

  it('refuses to export when no columns are selected', async () => {
    createComponent();
    await fixture.whenStable();
    await component.openFile();

    component.toggleColumn('id');
    component.toggleColumn('name');

    await component.exportData();

    expect(notificationMock.showError).toHaveBeenCalled();
    expect(wailsMock.saveTableExportDialog).not.toHaveBeenCalled();
  });

  it('exports the selected columns with the chosen delimiter', async () => {
    createComponent();
    await fixture.whenStable();
    await component.openFile();

    component.toggleColumn('name');
    state().delimiter.set('\t');

    await component.exportData();

    expect(wailsMock.saveTableExportDialog).toHaveBeenCalledWith('sample.tsv');
    expect(wailsMock.exportTableFile).toHaveBeenCalledWith(
      '/data/sample.parquet',
      '/data/sample.csv',
      ['id'],
      '\t'
    );
    expect(notificationMock.showSuccess).toHaveBeenCalled();
  });

  it('derives the default export name from a .csv source file', async () => {
    wailsMock.openTableFileDialog.mockResolvedValue('/data/sample.csv');
    wailsMock.getTableFileInfo.mockResolvedValue(sampleCSVInfo);
    createComponent();
    await fixture.whenStable();
    await component.openFile();

    await component.exportData();

    expect(wailsMock.saveTableExportDialog).toHaveBeenCalledWith('sample.csv');
  });

  it('does nothing when the export destination dialog is cancelled', async () => {
    wailsMock.saveTableExportDialog.mockResolvedValue('');
    createComponent();
    await fixture.whenStable();
    await component.openFile();

    await component.exportData();

    expect(wailsMock.exportTableFile).not.toHaveBeenCalled();
  });

  it('does not show an error when the file dialog is cancelled with a runtime error', async () => {
    wailsMock.openTableFileDialog.mockRejectedValue(
      new Error('{"message":"cancelled by user","cause":{},"kind":"RuntimeError"}')
    );
    createComponent();
    await fixture.whenStable();

    await component.openFile();

    expect(notificationMock.showError).not.toHaveBeenCalled();
  });

  it('does not show an error when the export dialog is cancelled with a runtime error', async () => {
    wailsMock.saveTableExportDialog.mockRejectedValue(
      new Error('{"message":"cancelled by user","cause":{},"kind":"RuntimeError"}')
    );
    createComponent();
    await fixture.whenStable();
    await component.openFile();

    await component.exportData();

    expect(notificationMock.showError).not.toHaveBeenCalled();
    expect(wailsMock.exportTableFile).not.toHaveBeenCalled();
  });

  it('closes the open file on destroy', async () => {
    createComponent();
    await fixture.whenStable();
    await component.openFile();

    await component.ngOnDestroy();

    expect(wailsMock.closeTableFile).toHaveBeenCalledWith('/data/sample.parquet');
  });

  it('updates export progress from the wails progress signal', async () => {
    createComponent();
    await fixture.whenStable();

    (wailsMock.progress as any).set({
      type: 'generic',
      id: 'table-export',
      message: 'Exported 50/100 rows',
      percentage: 50,
      status: 'in_progress'
    });
    fixture.detectChanges();

    expect(state().exportMessage()).toBe('Exported 50/100 rows');
    expect(state().exportPercentage()).toBe(50);
  });

  it('hides the row-groups stat for a non-parquet source file', async () => {
    wailsMock.openTableFileDialog.mockResolvedValue('/data/sample.csv');
    wailsMock.getTableFileInfo.mockResolvedValue(sampleCSVInfo);
    createComponent();
    await fixture.whenStable();
    await component.openFile();
    fixture.detectChanges();

    const text = fixture.nativeElement.textContent as string;
    expect(text).not.toContain('Row groups');
  });
});
