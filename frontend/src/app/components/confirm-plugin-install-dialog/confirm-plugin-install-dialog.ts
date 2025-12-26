import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';

export interface PluginInstallConfirmData {
  repo: string;
  ref?: string;
  name: string;
  id: string;
  version: string;
  author: string;
  description: string;
  category: string;
}

@Component({
  selector: 'app-confirm-plugin-install-dialog',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatDividerModule
  ],
  templateUrl: './confirm-plugin-install-dialog.html',
  styleUrl: './confirm-plugin-install-dialog.scss',
})
export class ConfirmPluginInstallDialog {
  constructor(
    public dialogRef: MatDialogRef<ConfirmPluginInstallDialog>,
    @Inject(MAT_DIALOG_DATA) public data: PluginInstallConfirmData
  ) {}

  confirm() {
    this.dialogRef.close(true);
  }

  cancel() {
    this.dialogRef.close(false);
  }
}
