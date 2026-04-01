import { Component, EventEmitter, OnInit, Output, signal, Inject, ChangeDetectionStrategy } from '@angular/core';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { FileHandler, ImportedFile } from '../../interfaces/file-handler.interface';
import { FILE_HANDLER } from '../../tokens/injection-tokens';

@Component({
  selector: 'cld-imported-file-selection',
  standalone: true,
  imports: [
    MatFormFieldModule,
    MatSelectModule
  ],
  templateUrl: './imported-file-selection.html',
  styleUrl: './imported-file-selection.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class ImportedFileSelection implements OnInit {
  files = signal<ImportedFile[]>([]);

  @Output() selected: EventEmitter<string> = new EventEmitter<string>();
  @Output() columns: EventEmitter<string[]> = new EventEmitter<string[]>();
  @Output() fileInfo: EventEmitter<ImportedFile> = new EventEmitter<ImportedFile>();

  constructor(@Inject(FILE_HANDLER) private fileHandler: FileHandler) {}

  async ngOnInit() {
    await this.loadFiles();
  }

  async loadFiles() {
    try {
      const files = await this.fileHandler.getImportedFiles();
      this.files.set(files);
    } catch (error) {
      this.files.set([]);
    }
  }

  async selectFile(file: ImportedFile) {
    this.selected.emit(file.path);
    this.fileInfo.emit(file);

    try {
      const preview = await this.fileHandler.parseDataFile(file.path);
      if (preview && preview.columns) {
        this.columns.emit(preview.columns);
      }
    } catch (error) {
      this.columns.emit([]);
    }
  }
}
