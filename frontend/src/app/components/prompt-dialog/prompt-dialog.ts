import { Component, Inject, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { FormsModule } from '@angular/forms';

export interface PromptDialogData {
  title: string;
  message?: string;
  label?: string;
  value?: string;
  placeholder?: string;
  confirmText?: string;
  cancelText?: string;
}

@Component({
  selector: 'app-prompt-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    FormsModule
  ],
  templateUrl: './prompt-dialog.html',
  styleUrl: './prompt-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class PromptDialogComponent {
  inputValue: string = '';

  constructor(
    public dialogRef: MatDialogRef<PromptDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: PromptDialogData
  ) {
    this.inputValue = data.value || '';
    this.data.confirmText = data.confirmText || 'OK';
    this.data.cancelText = data.cancelText || 'Cancel';
    this.data.label = data.label || 'Input';
  }

  onConfirm(): void {
    this.dialogRef.close(this.inputValue);
  }

  onCancel(): void {
    this.dialogRef.close(null);
  }
}