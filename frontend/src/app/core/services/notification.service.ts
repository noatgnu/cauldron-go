import { Injectable } from '@angular/core';
import { MatSnackBar, MatSnackBarConfig } from '@angular/material/snack-bar';

@Injectable({
  providedIn: 'root'
})
export class NotificationService {
  constructor(private snackBar: MatSnackBar) {}

  showError(message: string, duration: number = 5000): void {
    const config: MatSnackBarConfig = {
      duration: duration,
      horizontalPosition: 'center',
      verticalPosition: 'top',
      panelClass: ['error-snackbar']
    };
    this.snackBar.open(message, 'Close', config);
  }

  showSuccess(message: string, duration: number = 3000): void {
    const config: MatSnackBarConfig = {
      duration: duration,
      horizontalPosition: 'center',
      verticalPosition: 'top',
      panelClass: ['success-snackbar']
    };
    this.snackBar.open(message, 'Close', config);
  }

  showInfo(message: string, duration: number = 3000): void {
    const config: MatSnackBarConfig = {
      duration: duration,
      horizontalPosition: 'center',
      verticalPosition: 'top',
      panelClass: ['info-snackbar']
    };
    this.snackBar.open(message, 'Close', config);
  }

  showWarning(message: string, duration: number = 4000): void {
    const config: MatSnackBarConfig = {
      duration: duration,
      horizontalPosition: 'center',
      verticalPosition: 'top',
      panelClass: ['warning-snackbar']
    };
    this.snackBar.open(message, 'Close', config);
  }
}
