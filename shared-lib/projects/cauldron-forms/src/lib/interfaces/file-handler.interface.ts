export interface FileInfo {
  name: string;
  path: string;
  size: number;
  content?: string | ArrayBuffer;
}

export interface ImportedFile {
  id: number;
  name: string;
  path: string;
  size: number;
  importedAt: number;
  fileType: string;
  preview: string;
}

export interface FileHandler {
  readFile(path: string): Promise<string>;
  readFileAsUint8Array?(path: string): Promise<Uint8Array>;
  readFilePreview(path: string, lines: number): Promise<string>;
  openFileDialog(title: string, accept?: string): Promise<string | null>;
  openDirectoryDialog(title: string): Promise<string | null>;
  saveTempFile(filename: string, content: string | Uint8Array): Promise<string>;
  getFileHeaders(path: string): Promise<string[]>;
  getImportedFiles(): Promise<ImportedFile[]>;
  parseDataFile(path: string): Promise<{ columns: string[]; rows: string[][] }>;
}
