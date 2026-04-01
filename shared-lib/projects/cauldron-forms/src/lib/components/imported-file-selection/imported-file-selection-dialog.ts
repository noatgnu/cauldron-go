import { Component, ChangeDetectionStrategy } from '@angular/core';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { ImportedFileSelection } from './imported-file-selection';
import { ImportedFile } from '../../interfaces/file-handler.interface';

@Component({
  selector: 'cld-imported-file-selection-dialog',
  standalone: true,
  imports: [
    MatDialogModule,
    MatButtonModule,
    ImportedFileSelection
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <h2 mat-dialog-title id="file-selection-title">Select Imported File</h2>
    <mat-dialog-content aria-labelledby="file-selection-title">
      <cld-imported-file-selection
        (selected)="onFileSelected($event)"
        (columns)="onColumnsLoaded($event)"
        (fileInfo)="onFileInfo($event)">
      </cld-imported-file-selection>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button (click)="cancel()" aria-label="Cancel file selection">Cancel</button>
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
