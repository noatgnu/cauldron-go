import { Injectable } from '@angular/core';
import { FileHandler, FileInfo, ImportedFile } from '@cauldron/forms';
import { Wails } from '../services/wails';

@Injectable({ providedIn: 'root' })
export class WailsFileHandler implements FileHandler {
  constructor(private wails: Wails) {}

  async readFile(path: string): Promise<string> {
    return this.wails.readFile(path);
  }

  async readFileAsUint8Array(path: string): Promise<Uint8Array> {
    return this.wails.readFileAsUint8Array(path);
  }

  async readFilePreview(path: string, lines: number): Promise<string> {
    const preview = await this.wails.readFilePreview(path, lines);
    return preview.join('\n');
  }

  async openFileDialog(title: string, accept?: string): Promise<string | null> {
    const path = await this.wails.openFileDialog(title);
    return path || null;
  }

  async openDirectoryDialog(title: string): Promise<string | null> {
    const path = await this.wails.openDirectoryDialog(title);
    return path || null;
  }

  async saveTempFile(filename: string, content: string | Uint8Array): Promise<string> {
    const stringContent = typeof content === 'string' ? content : new TextDecoder().decode(content);
    return this.wails.saveTempFile(filename, stringContent);
  }

  async getFileHeaders(path: string): Promise<string[]> {
    const content = await this.wails.readFile(path);
    if (!content) return [];

    const lines = content.split('\n');
    if (lines.length === 0) return [];

    const firstLine = lines[0].trim();
    const delimiter = firstLine.includes('\t') ? '\t' : ',';
    return firstLine.split(delimiter).map(h => h.trim());
  }

  async getImportedFiles(): Promise<ImportedFile[]> {
    const files = await this.wails.getImportedFiles();
    return files.map(f => ({
      id: f.id,
      name: f.name,
      path: f.path,
      size: f.size,
      importedAt: f.importedAt,
      fileType: f.fileType,
      preview: f.preview
    }));
  }

  async parseDataFile(path: string): Promise<{ columns: string[]; rows: string[][] }> {
    const preview = await this.wails.parseDataFile(path, 10);
    return {
      columns: preview.headers || [],
      rows: preview.rows || []
    };
  }
}
