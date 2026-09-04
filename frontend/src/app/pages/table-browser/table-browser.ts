import { Component, OnDestroy, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatSelectModule } from '@angular/material/select';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatCardModule } from '@angular/material/card';
import { Wails, DataFileInfo } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

type Delimiter = ',' | '\t';

@Component({
  selector: 'app-table-browser',
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatToolbarModule,
    MatIconModule,
    MatTableModule,
    MatPaginatorModule,
    MatCheckboxModule,
    MatSelectModule,
    MatFormFieldModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    MatCardModule
  ],
  templateUrl: './table-browser.html',
  styleUrl: './table-browser.scss'
})
export class TableBrowser implements OnDestroy {
  private readonly wails = inject(Wails);
  private readonly notification = inject(NotificationService);

  protected filePath = signal<string | null>(null);
  protected fileInfo = signal<DataFileInfo | null>(null);
  protected rows = signal<Record<string, any>[]>([]);
  protected pageIndex = signal(0);
  protected pageSize = signal(25);
  protected loadingFile = signal(false);
  protected loadingPage = signal(false);
  protected selectedColumns = signal<Set<string>>(new Set());
  protected delimiter = signal<Delimiter>(',');
  protected exporting = signal(false);
  protected exportMessage = signal('');
  protected exportPercentage = signal(0);

  protected displayedColumns = computed(() => this.fileInfo()?.columns.map(c => c.name) ?? []);
  protected fileName = computed(() => {
    const path = this.filePath();
    if (!path) return '';
    return path.split(/[\\/]/).pop() ?? path;
  });

  constructor() {
    effect(() => {
      const progress = this.wails.progress();
      if (progress && progress.id === 'table-export') {
        this.exportMessage.set(progress.message);
        this.exportPercentage.set(progress.percentage);
        if (progress.status === 'completed' || progress.status === 'error') {
          this.exporting.set(false);
        }
      }
    });
  }

  async ngOnDestroy() {
    const path = this.filePath();
    if (path) {
      await this.wails.closeTableFile(path).catch(() => {});
    }
  }

  async openFile() {
    try {
      const path = await this.wails.openTableFileDialog();
      if (!path) return;
      await this.loadFile(path);
    } catch (error) {
      if (this.isDialogCancelled(error)) return;
      this.notification.showError(`Failed to open file dialog: ${error}`);
    }
  }

  private isDialogCancelled(error: unknown): boolean {
    return error instanceof Error && error.message.includes('cancelled by user');
  }

  private async loadFile(path: string) {
    const previous = this.filePath();
    if (previous && previous !== path) {
      await this.wails.closeTableFile(previous).catch(() => {});
    }

    this.loadingFile.set(true);
    try {
      const info = await this.wails.getTableFileInfo(path);
      this.filePath.set(path);
      this.fileInfo.set(info);
      this.selectedColumns.set(new Set(info?.columns.map(c => c.name) ?? []));
      this.pageIndex.set(0);
      await this.loadPage();
    } catch (error) {
      this.notification.showError(`Failed to open file: ${error}`);
    } finally {
      this.loadingFile.set(false);
    }
  }

  private async loadPage() {
    const path = this.filePath();
    if (!path) return;

    this.loadingPage.set(true);
    try {
      const offset = this.pageIndex() * this.pageSize();
      const page = await this.wails.getTableFilePage(path, offset, this.pageSize());
      this.rows.set(page || []);
    } catch (error) {
      this.notification.showError(`Failed to read page: ${error}`);
    } finally {
      this.loadingPage.set(false);
    }
  }

  async onPageChange(event: PageEvent) {
    this.pageIndex.set(event.pageIndex);
    this.pageSize.set(event.pageSize);
    await this.loadPage();
  }

  toggleColumn(name: string) {
    const current = new Set(this.selectedColumns());
    if (current.has(name)) {
      current.delete(name);
    } else {
      current.add(name);
    }
    this.selectedColumns.set(current);
  }

  isColumnSelected(name: string): boolean {
    return this.selectedColumns().has(name);
  }

  async exportData() {
    const path = this.filePath();
    const info = this.fileInfo();
    if (!path || !info) return;

    const columns = info.columns.map(c => c.name).filter(name => this.isColumnSelected(name));
    if (columns.length === 0) {
      this.notification.showError('Select at least one column to export');
      return;
    }

    const ext = this.delimiter() === '\t' ? 'tsv' : 'csv';
    const defaultName = this.fileName().replace(/\.(parquet|csv|tsv)$/i, `.${ext}`);

    let outputPath: string;
    try {
      outputPath = await this.wails.saveTableExportDialog(defaultName);
    } catch (error) {
      if (!this.isDialogCancelled(error)) {
        this.notification.showError(`Failed to open save dialog: ${error}`);
      }
      return;
    }
    if (!outputPath) return;

    try {
      this.exporting.set(true);
      this.exportMessage.set('Starting export...');
      this.exportPercentage.set(0);

      await this.wails.exportTableFile(path, outputPath, columns, this.delimiter());
      this.notification.showSuccess(`Exported to ${outputPath}`);
    } catch (error) {
      this.notification.showError(`Export failed: ${error}`);
    } finally {
      this.exporting.set(false);
    }
  }

  formatBytes(bytes: number): string {
    if (!bytes) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  }
}
