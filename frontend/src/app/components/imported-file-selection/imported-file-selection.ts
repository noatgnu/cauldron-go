import {Component, EventEmitter, OnInit, Output, signal} from '@angular/core';
import {Wails, ImportedFile} from "../../core/services/wails";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatSelectModule} from "@angular/material/select";

@Component({
  selector: 'app-imported-file-selection',
  imports: [
    MatFormFieldModule,
    MatSelectModule
  ],
  templateUrl: './imported-file-selection.html',
  styleUrl: './imported-file-selection.scss',
})
export class ImportedFileSelection implements OnInit {
  files = signal<ImportedFile[]>([]);

  @Output() selected: EventEmitter<string> = new EventEmitter<string>();
  @Output() columns: EventEmitter<string[]> = new EventEmitter<string[]>();
  @Output() fileInfo: EventEmitter<ImportedFile> = new EventEmitter<ImportedFile>();

  constructor(private wails: Wails) {}

  async ngOnInit() {
    await this.loadFiles();
  }

  async loadFiles() {
    try {
      const files = await this.wails.getImportedFiles();
      this.files.set(files);
    } catch (error) {
      console.error('Failed to load imported files:', error);
    }
  }

  async selectFile(file: ImportedFile) {
    this.selected.emit(file.path);
    this.fileInfo.emit(file);

    try {
      const preview = await this.wails.parseDataFile(file.path, 1);
      if (preview && preview.headers) {
        this.columns.emit(preview.headers);
      }
    } catch (error) {
      console.error('Failed to parse file:', error);
    }
  }
}
