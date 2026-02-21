export interface NotificationHandler {
  showError(message: string, duration?: number): void;
  showSuccess(message: string, duration?: number): void;
  showWarning(message: string, duration?: number): void;
  showInfo(message: string, duration?: number): void;
}
