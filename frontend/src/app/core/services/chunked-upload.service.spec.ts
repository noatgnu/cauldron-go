import { TestBed } from '@angular/core/testing';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { ChunkedUploadService, arrayBufferToBase64, deriveUploadId, DEFAULT_UPLOAD_CHUNK_SIZE } from './chunked-upload.service';
import { Wails } from './wails';

describe('arrayBufferToBase64', () => {
  it('round-trips small buffers', () => {
    const bytes = new Uint8Array([72, 101, 108, 108, 111]);
    const encoded = arrayBufferToBase64(bytes.buffer);
    expect(atob(encoded)).toBe('Hello');
  });

  it('round-trips buffers larger than the base64 encoding block size', () => {
    const size = 100_000;
    const bytes = new Uint8Array(size);
    for (let i = 0; i < size; i++) bytes[i] = i % 256;

    const encoded = arrayBufferToBase64(bytes.buffer);
    const decoded = atob(encoded);
    expect(decoded.length).toBe(size);
    for (let i = 0; i < size; i++) {
      expect(decoded.charCodeAt(i)).toBe(i % 256);
    }
  });
});

describe('deriveUploadId', () => {
  it('is stable for the same file identity', () => {
    const a = new File(['data'], 'sample.txt', { lastModified: 1000 });
    const b = new File(['data'], 'sample.txt', { lastModified: 1000 });
    expect(deriveUploadId(a)).toBe(deriveUploadId(b));
  });

  it('differs when filename, size, or lastModified differ', () => {
    const base = new File(['data'], 'sample.txt', { lastModified: 1000 });
    const differentName = new File(['data'], 'other.txt', { lastModified: 1000 });
    const differentContent = new File(['data-longer'], 'sample.txt', { lastModified: 1000 });
    const differentModified = new File(['data'], 'sample.txt', { lastModified: 2000 });

    const baseId = deriveUploadId(base);
    expect(deriveUploadId(differentName)).not.toBe(baseId);
    expect(deriveUploadId(differentContent)).not.toBe(baseId);
    expect(deriveUploadId(differentModified)).not.toBe(baseId);
  });
});

describe('ChunkedUploadService', () => {
  let service: ChunkedUploadService;
  let wailsMock: any;

  beforeEach(() => {
    wailsMock = {
      startChunkedUpload: vi.fn().mockResolvedValue([]),
      writeChunk: vi.fn().mockResolvedValue(undefined),
      getReceivedUploadChunks: vi.fn(),
      completeChunkedUpload: vi.fn().mockResolvedValue('/staged/path/sample.bin'),
      abortChunkedUpload: vi.fn().mockResolvedValue(undefined)
    };

    TestBed.configureTestingModule({
      providers: [
        ChunkedUploadService,
        { provide: Wails, useValue: wailsMock }
      ]
    });

    service = TestBed.inject(ChunkedUploadService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('splits a file into chunks of the requested size and uploads each one', async () => {
    const content = 'a'.repeat(10);
    const file = new File([content], 'sample.bin', { lastModified: 42 });

    const path = await service.upload(file, { chunkSize: 4, uploadId: 'upload-1' });

    expect(wailsMock.startChunkedUpload).toHaveBeenCalledWith('upload-1', 'sample.bin', 3);
    expect(wailsMock.writeChunk).toHaveBeenCalledTimes(3);
    expect(wailsMock.writeChunk.mock.calls[0][0]).toBe('upload-1');
    expect(wailsMock.writeChunk.mock.calls[0][1]).toBe(0);
    expect(wailsMock.writeChunk.mock.calls[1][1]).toBe(1);
    expect(wailsMock.writeChunk.mock.calls[2][1]).toBe(2);
    expect(wailsMock.completeChunkedUpload).toHaveBeenCalledWith('upload-1');
    expect(path).toBe('/staged/path/sample.bin');
  });

  it('never reads the whole file into memory: each chunk argument stays within the chunk size', async () => {
    const content = 'x'.repeat(10);
    const file = new File([content], 'sample.bin');

    await service.upload(file, { chunkSize: 4, uploadId: 'upload-1' });

    for (const call of wailsMock.writeChunk.mock.calls) {
      const base64Chunk: string = call[2];
      const decodedLength = atob(base64Chunk).length;
      expect(decodedLength).toBeLessThanOrEqual(4);
    }
  });

  it('skips chunks the server already has when resuming', async () => {
    const content = 'a'.repeat(10);
    const file = new File([content], 'sample.bin');
    wailsMock.startChunkedUpload.mockResolvedValue([0, 1]);

    await service.upload(file, { chunkSize: 4, uploadId: 'upload-1' });

    expect(wailsMock.writeChunk).toHaveBeenCalledTimes(1);
    expect(wailsMock.writeChunk.mock.calls[0][1]).toBe(2);
  });

  it('reports progress after each chunk, including skipped ones', async () => {
    const content = 'a'.repeat(10);
    const file = new File([content], 'sample.bin');
    wailsMock.startChunkedUpload.mockResolvedValue([0]);

    const progressUpdates: any[] = [];
    await service.upload(file, {
      chunkSize: 4,
      uploadId: 'upload-1',
      onProgress: (p) => progressUpdates.push(p)
    });

    expect(progressUpdates).toHaveLength(3);
    expect(progressUpdates[0]).toEqual({ uploadId: 'upload-1', sentChunks: 1, totalChunks: 3, bytesSent: 4, totalBytes: 10 });
    expect(progressUpdates[2]).toEqual({ uploadId: 'upload-1', sentChunks: 3, totalChunks: 3, bytesSent: 10, totalBytes: 10 });
  });

  it('derives a default upload id and chunk size when none are given', async () => {
    const file = new File(['small file'], 'sample.bin', { lastModified: 5 });

    await service.upload(file);

    expect(wailsMock.startChunkedUpload).toHaveBeenCalledWith(deriveUploadId(file), 'sample.bin', 1);
  });

  it('uploads a zero-byte file as a single empty chunk', async () => {
    const file = new File([], 'empty.bin');

    await service.upload(file, { uploadId: 'upload-empty' });

    expect(wailsMock.startChunkedUpload).toHaveBeenCalledWith('upload-empty', 'empty.bin', 1);
    expect(wailsMock.writeChunk).toHaveBeenCalledTimes(1);
  });

  it('delegates abort to the underlying Wails binding', async () => {
    await service.abort('upload-1');
    expect(wailsMock.abortChunkedUpload).toHaveBeenCalledWith('upload-1');
  });

  it('exposes a default chunk size under the backend per-chunk cap', () => {
    expect(DEFAULT_UPLOAD_CHUNK_SIZE).toBeLessThanOrEqual(8 * 1024 * 1024);
  });
});
