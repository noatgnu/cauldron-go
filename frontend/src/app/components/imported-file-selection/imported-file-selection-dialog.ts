import { Component } from '@angular/core';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { ImportedFileSelection } from './imported-file-selection';
import { ImportedFile } from '../../core/services/wails';

@Component({
  selector: 'app-imported-file-selection-dialog',
  imports: [
    MatDialogModule,
    MatButtonModule,
    ImportedFileSelection
  ],
  template: `
    <h2 mat-dialog-title>Select Imported File</h2>
    <mat-dialog-content>
      <app-imported-file-selection
        (selected)="onFileSelected($event)"
        (columns)="onColumnsLoaded($event)"
        (fileInfo)="onFileInfo($event)">
      </app-imported-file-selection>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button (click)="cancel()">Cancel</button>
    </mat-dialog-actions>
  `,
  styles: [`
    mat-dialog-content {
      min-width: 500px;
      padding: 20px 24px;
    }
  `]
})
export class ImportedFileSelectionDialog {
  private selectedFilePath?: string;
  private selectedColumns?: string[];
  private selectedFileInfo?: ImportedFile;

  constructor(
    private dialogRef: MatDialogRef<ImportedFileSelectionDialog>
  ) {}

  onFileSelected(filePath: string) {
    this.selectedFilePath = filePath;
  }

  onColumnsLoaded(columns: string[]) {
    this.selectedColumns = columns;
    if (this.selectedFilePath) {
      this.dialogRef.close({
        filePath: this.selectedFilePath,
        columns: this.selectedColumns,
        fileInfo: this.selectedFileInfo
      });
    }
  }

  onFileInfo(fileInfo: ImportedFile) {
    this.selectedFileInfo = fileInfo;
  }

  cancel() {
    this.dialogRef.close();
  }
}
