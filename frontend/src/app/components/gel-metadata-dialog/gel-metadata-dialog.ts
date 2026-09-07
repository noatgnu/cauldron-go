import { Component, Inject, ChangeDetectionStrategy, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

export interface GelMetadataDialogData {
  metadata: Partial<Record<string, string>>;
}

@Component({
  selector: 'app-gel-metadata-dialog',
  imports: [CommonModule, MatDialogModule, MatButtonModule, MatIconModule],
  styleUrl: './gel-metadata-dialog.scss',
  templateUrl: './gel-metadata-dialog.html',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class GelMetadataDialog {
  protected entries = signal<[string, string][]>([]);

  constructor(
    public dialogRef: MatDialogRef<GelMetadataDialog>,
    @Inject(MAT_DIALOG_DATA) public data: GelMetadataDialogData
  ) {
    const sorted = Object.entries(data.metadata).sort(([a], [b]) => a.localeCompare(b)) as [string, string][];
    this.entries.set(sorted);
  }

  onClose(): void {
    this.dialogRef.close();
  }
}
