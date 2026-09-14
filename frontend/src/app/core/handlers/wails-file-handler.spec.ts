import { TestBed } from '@angular/core/testing';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { WailsFileHandler } from './wails-file-handler';
import { Wails } from '../services/wails';
import { FilePickerService } from '../services/file-picker.service';

describe('WailsFileHandler', () => {
  let handler: WailsFileHandler;
  let wailsMock: any;
  let filePickerMock: any;

  beforeEach(() => {
    wailsMock = {
      openFileDialog: vi.fn().mockResolvedValue('/native/path.tsv'),
      openDirectoryDialog: vi.fn().mockResolvedValue('/native/dir'),
      getRuntimeCapabilities: vi.fn().mockResolvedValue({ serverMode: false, nativeFileAccess: true, maxUploadChunkBytes: 8388608 })
    };
    filePickerMock = {
      pickFilePath: vi.fn().mockResolvedValue('/resolved/path.tsv')
    };

    TestBed.configureTestingModule({
      providers: [
        WailsFileHandler,
        { provide: Wails, useValue: wailsMock },
        { provide: FilePickerService, useValue: filePickerMock }
      ]
    });

    handler = TestBed.inject(WailsFileHandler);
  });

  describe('openFileDialog', () => {
    it('delegates to FilePickerService with the native dialog and accept filter', async () => {
      const result = await handler.openFileDialog('Select File', '.tsv');

      expect(filePickerMock.pickFilePath).toHaveBeenCalledWith(expect.any(Function), '.tsv');
      expect(result).toBe('/resolved/path.tsv');

      const nativeDialogFn = filePickerMock.pickFilePath.mock.calls[0][0];
      await nativeDialogFn();
      expect(wailsMock.openFileDialog).toHaveBeenCalledWith('Select File');
    });
  });

  describe('openDirectoryDialog', () => {
    it('uses the native dialog when native file access is available', async () => {
      const result = await handler.openDirectoryDialog('Select Directory');
      expect(wailsMock.openDirectoryDialog).toHaveBeenCalledWith('Select Directory');
      expect(result).toBe('/native/dir');
    });

    it('throws when native file access is unavailable', async () => {
      wailsMock.getRuntimeCapabilities.mockResolvedValue({ serverMode: true, nativeFileAccess: false, maxUploadChunkBytes: 8388608 });

      await expect(handler.openDirectoryDialog('Select Directory')).rejects.toThrow(/server mode/i);
      expect(wailsMock.openDirectoryDialog).not.toHaveBeenCalled();
    });
  });
});
