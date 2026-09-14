import { TestBed } from '@angular/core/testing';
import { vi, describe, it, expect, afterEach } from 'vitest';
import { FilePickerService } from './file-picker.service';
import { Wails } from './wails';
import { ChunkedUploadService } from './chunked-upload.service';

async function waitForFileInput(): Promise<HTMLInputElement> {
  for (let attempt = 0; attempt < 10; attempt++) {
    const input = document.querySelector('input[type="file"]') as HTMLInputElement | null;
    if (input) return input;
    await Promise.resolve();
  }
  throw new Error('expected a file input to be appended to the document');
}

describe('FilePickerService', () => {
  let wailsMock: any;
  let chunkedUploadMock: any;

  function createService(): FilePickerService {
    TestBed.configureTestingModule({
      providers: [
        FilePickerService,
        { provide: Wails, useValue: wailsMock },
        { provide: ChunkedUploadService, useValue: chunkedUploadMock }
      ]
    });
    return TestBed.inject(FilePickerService);
  }

  afterEach(() => {
    document.querySelectorAll('input[type="file"]').forEach(el => el.remove());
  });

  it('calls the native dialog when native file access is available', async () => {
    wailsMock = { getRuntimeCapabilities: vi.fn().mockResolvedValue({ serverMode: false, nativeFileAccess: true, maxUploadChunkBytes: 8388608 }) };
    chunkedUploadMock = { upload: vi.fn() };
    const service = createService();
    const openNativeDialog = vi.fn().mockResolvedValue('/native/gel.tiff');

    const result = await service.pickFilePath(openNativeDialog, '.tif,.tiff');

    expect(openNativeDialog).toHaveBeenCalled();
    expect(chunkedUploadMock.upload).not.toHaveBeenCalled();
    expect(result).toBe('/native/gel.tiff');
  });

  it('returns null when the native dialog is cancelled', async () => {
    wailsMock = { getRuntimeCapabilities: vi.fn().mockResolvedValue({ serverMode: false, nativeFileAccess: true, maxUploadChunkBytes: 8388608 }) };
    chunkedUploadMock = { upload: vi.fn() };
    const service = createService();
    const openNativeDialog = vi.fn().mockResolvedValue('');

    const result = await service.pickFilePath(openNativeDialog);

    expect(result).toBeNull();
  });

  it('uploads a browser-picked file when native file access is unavailable', async () => {
    wailsMock = { getRuntimeCapabilities: vi.fn().mockResolvedValue({ serverMode: true, nativeFileAccess: false, maxUploadChunkBytes: 8388608 }) };
    chunkedUploadMock = { upload: vi.fn().mockResolvedValue('/staged/gel.tiff') };
    const service = createService();
    const openNativeDialog = vi.fn();

    const resultPromise = service.pickFilePath(openNativeDialog, '.tif,.tiff');
    const input = await waitForFileInput();
    expect(input.accept).toBe('.tif,.tiff');
    const file = new File(['bytes'], 'gel.tiff');
    Object.defineProperty(input, 'files', { value: [file], configurable: true });
    input.dispatchEvent(new Event('change'));

    const result = await resultPromise;
    expect(openNativeDialog).not.toHaveBeenCalled();
    expect(chunkedUploadMock.upload).toHaveBeenCalledWith(file);
    expect(result).toBe('/staged/gel.tiff');
  });

  it('returns null when the browser file pick is cancelled', async () => {
    wailsMock = { getRuntimeCapabilities: vi.fn().mockResolvedValue({ serverMode: true, nativeFileAccess: false, maxUploadChunkBytes: 8388608 }) };
    chunkedUploadMock = { upload: vi.fn() };
    const service = createService();

    const resultPromise = service.pickFilePath(vi.fn());
    const input = await waitForFileInput();
    input.dispatchEvent(new Event('cancel'));

    const result = await resultPromise;
    expect(chunkedUploadMock.upload).not.toHaveBeenCalled();
    expect(result).toBeNull();
  });
});
