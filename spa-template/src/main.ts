import { bootstrapApplication } from '@angular/platform-browser';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { FILE_HANDLER, NOTIFICATION_HANDLER, LOG_HANDLER } from '@cauldron/forms';
import { AppComponent } from './app/app';
import { BrowserFileHandler } from './app/services/browser-file-handler';
import { BrowserNotificationHandler } from './app/services/browser-notification-handler';
import { BrowserLogHandler } from './app/services/browser-log-handler';

bootstrapApplication(AppComponent, {
  providers: [
    provideAnimationsAsync(),
    { provide: FILE_HANDLER, useClass: BrowserFileHandler },
    { provide: NOTIFICATION_HANDLER, useClass: BrowserNotificationHandler },
    { provide: LOG_HANDLER, useClass: BrowserLogHandler }
  ]
}).catch(err => {
  const logHandler = new BrowserLogHandler();
  logHandler.error('Bootstrap failed: ' + err.message);
});
