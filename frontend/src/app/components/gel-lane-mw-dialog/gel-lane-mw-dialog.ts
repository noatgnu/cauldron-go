import { Component, Inject, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { FormsModule } from '@angular/forms';

export interface GelLaneMwDialogData {
  laneLabel: string;
  markerMWs?: number[];
}

@Component({
  selector: 'app-gel-lane-mw-dialog',
  standalone: true,
  imports: [CommonModule, MatDialogModule, MatButtonModule, MatFormFieldModule, MatInputModule, FormsModule],
  templateUrl: './gel-lane-mw-dialog.html',
  styleUrl: './gel-lane-mw-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class GelLaneMwDialog {
  inputValue = signal('');
  error = signal('');

  constructor(
    public dialogRef: MatDialogRef<GelLaneMwDialog>,
    @Inject(MAT_DIALOG_DATA) public data: GelLaneMwDialogData
  ) {
    this.inputValue.set((data.markerMWs ?? []).join(', '));
  }

  onInputChange(value: string): void {
    this.inputValue.set(value);
    this.error.set('');
  }

  private parse(): number[] | null {
    const parts = this.inputValue()
      .split(',')
      .map(p => p.trim())
      .filter(p => p.length > 0);

    if (parts.length === 0) {
      this.error.set('Enter at least one molecular weight (kDa), separated by commas.');
      return null;
    }

    const values: number[] = [];
    for (const part of parts) {
      const n = Number(part);
      if (!isFinite(n) || n <= 0) {
        this.error.set(`"${part}" is not a valid positive number.`);
        return null;
      }
      values.push(n);
    }
    return values;
  }

  onConfirm(): void {
    const values = this.parse();
    if (values === null) return;
    this.dialogRef.close(values);
  }

  onCancel(): void {
    this.dialogRef.close(null);
  }
}
