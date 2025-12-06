import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTableModule } from '@angular/material/table';
import { MatIconModule } from '@angular/material/icon';
import { Wails, DataFilePreview } from '../../core/services/wails';

@Component({
  selector: 'app-import-dialog',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatTableModule,
    MatIconModule
  ],
  templateUrl: './import-dialog.html',
  styleUrl: './import-dialog.scss'
})
export class ImportDialog {
  protected selectedFile = signal('');
  protected preview = signal<DataFilePreview | null>(null);
  protected loading = signal(false);
  protected importing = signal(false);
  protected error = signal('');

  constructor(
    private wails: Wails,
    private dialogRef: MatDialogRef<ImportDialog>
  ) {}

  async selectFile(): Promise<void> {
    this.error.set('');
    this.loading.set(true);

    try {
      const filePath = await this.wails.openDataFileDialog();
      if (!filePath) {
        this.loading.set(false);
        return;
      }

      this.selectedFile.set(filePath);
      const preview = await this.wails.parseDataFile(filePath, 10);
      this.preview.set(preview);
    } catch (err: any) {
      this.error.set(err.message || 'Failed to load file');
    } finally {
      this.loading.set(false);
    }
  }

  async importFile(): Promise<void> {
    if (!this.selectedFile()) return;

    this.importing.set(true);
    this.error.set('');

    try {
      const fileId = await this.wails.importDataFile(this.selectedFile());
      this.dialogRef.close({ imported: true, fileId });
    } catch (err: any) {
      this.error.set(err.message || 'Failed to import file');
      this.importing.set(false);
    }
  }

  getDisplayedColumns(): string[] {
    const preview = this.preview();
    if (!preview || preview.headers.length === 0) return [];
    return preview.headers.slice(0, 5);
  }

  cancel(): void {
    this.dialogRef.close({ imported: false });
  }
}
