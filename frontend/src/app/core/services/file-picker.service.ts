import { Injectable } from '@angular/core';
import { Wails } from './wails';
import { ChunkedUploadService } from './chunked-upload.service';
import { pickBrowserFile } from './browser-file-picker';

/**
 * Resolves a server-local file path from either a native OS dialog (desktop) or a
 * browser file picker + chunked upload (server mode, no shared filesystem).
 */
@Injectable({
  providedIn: 'root'
})
export class FilePickerService {
  constructor(private wails: Wails, private chunkedUpload: ChunkedUploadService) {}

  async pickFilePath(openNativeDialog: () => Promise<string>, accept?: string): Promise<string | null> {
    const capabilities = await this.wails.getRuntimeCapabilities();
    if (capabilities.nativeFileAccess) {
      const path = await openNativeDialog();
      return path || null;
    }

    const file = await pickBrowserFile(accept);
    if (!file) return null;
    return this.chunkedUpload.upload(file);
  }
}
