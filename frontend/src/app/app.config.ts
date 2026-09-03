import { ApplicationConfig, importProvidersFrom, provideBrowserGlobalErrorListeners, provideZonelessChangeDetection } from '@angular/core';
import { provideRouter, withHashLocation } from '@angular/router';
import { MATERIAL_ANIMATIONS } from '@angular/material/core';
import * as PlotlyJS from 'plotly.js-dist-min';
import { PlotlyModule } from 'angular-plotly.js';
import {
  FILE_HANDLER,
  NOTIFICATION_HANDLER,
  LOG_HANDLER,
  COLOR_PALETTE_PROVIDER
} from '@cauldron/forms';
import { WailsFileHandler } from './core/handlers/wails-file-handler';
import { WailsNotificationHandler } from './core/handlers/wails-notification-handler';
import { WailsLogHandler } from './core/handlers/wails-log-handler';
import { WailsColorPaletteProvider } from './core/handlers/wails-color-palette-provider';

import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZonelessChangeDetection(),
    { provide: MATERIAL_ANIMATIONS, useValue: { animationsDisabled: false } },
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes, withHashLocation()),
    importProvidersFrom(PlotlyModule.forRoot(PlotlyJS)),
    { provide: FILE_HANDLER, useClass: WailsFileHandler },
    { provide: NOTIFICATION_HANDLER, useClass: WailsNotificationHandler },
    { provide: LOG_HANDLER, useClass: WailsLogHandler },
    { provide: COLOR_PALETTE_PROVIDER, useClass: WailsColorPaletteProvider }
  ]
};
