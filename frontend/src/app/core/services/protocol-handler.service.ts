import { Injectable } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { HandleProtocolURL } from '../../../wailsjs/go/main/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';

@Injectable({
  providedIn: 'root'
})
export class ProtocolHandlerService {
  constructor(private snackBar: MatSnackBar) {
    this.setupListeners();
  }

  private setupListeners(): void {
    if (window.runtime) {
      EventsOn('protocol:success', (data: { url: string }) => {
        console.log('[ProtocolHandler] Protocol URL handled successfully:', data.url);
      });

      EventsOn('protocol:error', (data: { url: string; error: string }) => {
        console.error('[ProtocolHandler] Protocol URL handling failed:', data.url, data.error);
        this.snackBar.open(`Failed to handle protocol URL: ${data.error}`, 'Close', {
          duration: 5000,
          horizontalPosition: 'center',
          verticalPosition: 'top',
          panelClass: ['error-snackbar']
        });
      });

      EventsOn('plugin:installed', () => {
        console.log('[ProtocolHandler] Plugin installed via protocol handler');
      });
    }
  }

  async handleURL(url: string): Promise<void> {
    try {
      await HandleProtocolURL(url);
    } catch (error) {
      console.error('[ProtocolHandler] Error handling protocol URL:', error);
      throw error;
    }
  }
}
