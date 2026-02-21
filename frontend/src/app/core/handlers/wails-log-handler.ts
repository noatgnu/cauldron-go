import { Injectable } from '@angular/core';
import { LogHandler } from '@cauldron/forms';
import { Wails } from '../services/wails';

@Injectable({ providedIn: 'root' })
export class WailsLogHandler implements LogHandler {
  constructor(private wails: Wails) {}

  async log(message: string): Promise<void> {
    if (this.wails.isWails) {
      await this.wails.logToFile(`[INFO] ${message}`);
    }
  }

  async error(message: string): Promise<void> {
    if (this.wails.isWails) {
      await this.wails.logToFile(`[ERROR] ${message}`);
    }
  }

  async warn(message: string): Promise<void> {
    if (this.wails.isWails) {
      await this.wails.logToFile(`[WARN] ${message}`);
    }
  }

  async debug(message: string): Promise<void> {
    if (this.wails.isWails) {
      await this.wails.logToFile(`[DEBUG] ${message}`);
    }
  }
}
