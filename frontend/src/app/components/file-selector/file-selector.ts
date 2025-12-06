import { Component, OnInit, signal, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Wails, ImportedFile } from '../../core/services/wails';

@Component({
  selector: 'app-file-selector',
  imports: [
    CommonModule,
    FormsModule,
    MatFormFieldModule,
    MatSelectModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule
  ],
  templateUrl: './file-selector.html',
  styleUrl: './file-selector.scss'
})
export class FileSelector implements OnInit {
  label = input<string>('Select Data File');
  placeholder = input<string>('Choose from imported files or browse');
  selectedPath = input<string>('');

  fileSelected = output<string>();

  protected importedFiles = signal<ImportedFile[]>([]);
  protected loading = signal(false);
  protected currentSelection = signal('');

  constructor(private wails: Wails) {}

  async ngOnInit() {
    await this.loadImportedFiles();
    if (this.selectedPath()) {
      this.currentSelection.set(this.selectedPath());
    }

    if (window.runtime) {
      window.runtime.EventsOn('file:imported', () => {
        this.loadImportedFiles();
      });
    }
  }

  async loadImportedFiles() {
    this.loading.set(true);
    try {
      const files = await this.wails.getImportedFiles();
      this.importedFiles.set(files);
    } catch (error) {
      console.error('Failed to load imported files:', error);
    } finally {
      this.loading.set(false);
    }
  }

  onSelectionChange(path: string): void {
    this.currentSelection.set(path);
    this.fileSelected.emit(path);
  }

  async browseFile(): Promise<void> {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        await this.wails.importDataFile(path);
        this.currentSelection.set(path);
        this.fileSelected.emit(path);
      }
    } catch (error) {
      console.error('Failed to browse file:', error);
    }
  }

  getFileDisplayName(file: ImportedFile): string {
    return `${file.name} (${file.preview})`;
  }
}
