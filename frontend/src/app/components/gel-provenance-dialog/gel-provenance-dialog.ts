import { Component, Inject, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { GelAnalysisProvenance } from '../../core/services/wails';

export interface GelProvenanceDialogData {
  provenance: GelAnalysisProvenance;
}

@Component({
  selector: 'app-gel-provenance-dialog',
  imports: [CommonModule, MatDialogModule, MatButtonModule, MatIconModule],
  styleUrl: './gel-provenance-dialog.scss',
  templateUrl: './gel-provenance-dialog.html',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class GelProvenanceDialog {
  constructor(
    public dialogRef: MatDialogRef<GelProvenanceDialog>,
    @Inject(MAT_DIALOG_DATA) public data: GelProvenanceDialogData
  ) {}

  onClose(): void {
    this.dialogRef.close();
  }
}
