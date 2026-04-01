import { Injectable } from '@angular/core';
import { Events } from '@wailsio/runtime';
import { NotificationService } from './notification.service';
import { HandleProtocolURL } from '../../../../bindings/github.com/noatgnu/cauldron-go/app';

@Injectable({
  providedIn: 'root'
})
export class ProtocolHandlerService {
  constructor(private notification: NotificationService) {
    this.setupListeners();
  }

  private setupListeners(): void {
    Events.On('protocol:success', (ev: any) => {
      const data = ev.data as { url: string };
      console.log('[ProtocolHandler] Protocol URL handled successfully:', data.url);
    });

    Events.On('protocol:error', (ev: any) => {
      const data = ev.data as { url: string; error: string };
      console.error('[ProtocolHandler] Protocol URL handling failed:', data.url, data.error);
      this.notification.showError(`Failed to handle protocol URL: ${data.error}`);
    });

    Events.On('plugin:installed', () => {
      console.log('[ProtocolHandler] Plugin installed via protocol handler');
    });
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
