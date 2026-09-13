import { Injectable } from '@angular/core';
import { Wails } from './wails';

/** 4MB, well under the backend's 8MB per-chunk cap. */
export const DEFAULT_UPLOAD_CHUNK_SIZE = 4 * 1024 * 1024;

/** btoa() over a huge string overflows the call stack; encode in 32KB sub-chunks instead. */
const BASE64_ENCODE_BLOCK_SIZE = 32 * 1024;

export interface ChunkedUploadProgress {
  uploadId: string;
  sentChunks: number;
  totalChunks: number;
  bytesSent: number;
  totalBytes: number;
}

export interface ChunkedUploadOptions {
  chunkSize?: number;
  uploadId?: string;
  onProgress?: (progress: ChunkedUploadProgress) => void;
}

export function readBlobAsArrayBuffer(blob: Blob): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as ArrayBuffer);
    reader.onerror = () => reject(reader.error ?? new Error('failed to read file chunk'));
    reader.readAsArrayBuffer(blob);
  });
}

export function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let offset = 0; offset < bytes.length; offset += BASE64_ENCODE_BLOCK_SIZE) {
    const block = bytes.subarray(offset, offset + BASE64_ENCODE_BLOCK_SIZE);
    binary += String.fromCharCode(...block);
  }
  return btoa(binary);
}

/** Stable across reloads for the same file, so an interrupted upload can be resumed by re-deriving the same ID. */
export function deriveUploadId(file: File): string {
  const key = `${file.name}:${file.size}:${file.lastModified}`;
  let hash = 0x811c9dc5;
  for (let i = 0; i < key.length; i++) {
    hash ^= key.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return `upload-${(hash >>> 0).toString(16)}`;
}

/**
 * Uploads a File to the backend in bounded-size chunks read via File.slice(), so the
 * full file is never held in memory. Resumable: re-calling upload() with the same
 * uploadId (or letting it re-derive one from the same file) skips chunks already received.
 */
@Injectable({
  providedIn: 'root'
})
export class ChunkedUploadService {
  constructor(private wails: Wails) {}

  async upload(file: File, options: ChunkedUploadOptions = {}): Promise<string> {
    const chunkSize = options.chunkSize ?? DEFAULT_UPLOAD_CHUNK_SIZE;
    const uploadId = options.uploadId ?? deriveUploadId(file);
    const totalChunks = Math.max(1, Math.ceil(file.size / chunkSize));

    const received = await this.wails.startChunkedUpload(uploadId, file.name, totalChunks);
    const receivedSet = new Set(received);

    for (let index = 0; index < totalChunks; index++) {
      const start = index * chunkSize;
      const end = Math.min(start + chunkSize, file.size);

      if (!receivedSet.has(index)) {
        const slice = file.slice(start, end);
        const buffer = await readBlobAsArrayBuffer(slice);
        await this.wails.writeChunk(uploadId, index, arrayBufferToBase64(buffer));
      }

      options.onProgress?.({
        uploadId,
        sentChunks: index + 1,
        totalChunks,
        bytesSent: end,
        totalBytes: file.size
      });
    }

    return this.wails.completeChunkedUpload(uploadId);
  }

  async abort(uploadId: string): Promise<void> {
    return this.wails.abortChunkedUpload(uploadId);
  }
}
