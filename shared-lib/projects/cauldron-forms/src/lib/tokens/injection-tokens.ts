import { InjectionToken } from '@angular/core';
import { FileHandler } from '../interfaces/file-handler.interface';
import { NotificationHandler } from '../interfaces/notification.interface';
import { LogHandler } from '../interfaces/log.interface';

export const FILE_HANDLER = new InjectionToken<FileHandler>('FILE_HANDLER');
export const NOTIFICATION_HANDLER = new InjectionToken<NotificationHandler>('NOTIFICATION_HANDLER');
export const LOG_HANDLER = new InjectionToken<LogHandler>('LOG_HANDLER');
