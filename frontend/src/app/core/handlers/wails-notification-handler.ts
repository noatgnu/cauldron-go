import { Injectable } from '@angular/core';
import { NotificationHandler } from '@cauldron/forms';
import { NotificationService } from '../services/notification.service';

@Injectable({ providedIn: 'root' })
export class WailsNotificationHandler implements NotificationHandler {
  constructor(private notificationService: NotificationService) {}

  showError(message: string, duration?: number): void {
    this.notificationService.showError(message, duration);
  }

  showSuccess(message: string, duration?: number): void {
    this.notificationService.showSuccess(message, duration);
  }

  showWarning(message: string, duration?: number): void {
    this.notificationService.showWarning(message, duration);
  }

  showInfo(message: string, duration?: number): void {
    this.notificationService.showInfo(message, duration);
  }
}
