import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { UpdateInfo } from '../../core/services/wails';

@Component({
  selector: 'app-update-available-dialog',
  imports: [CommonModule, MatDialogModule, MatButtonModule, MatIconModule],
  templateUrl: './update-available-dialog.html',
  styleUrl: './update-available-dialog.scss'
})
export class UpdateAvailableDialog {
  constructor(
    public dialogRef: MatDialogRef<UpdateAvailableDialog>,
    @Inject(MAT_DIALOG_DATA) public data: UpdateInfo
  ) {}

  viewRelease() {
    if (this.data.releaseUrl) {
      window.open(this.data.releaseUrl, '_blank');
    }
  }

  close() {
    this.dialogRef.close();
  }
}
