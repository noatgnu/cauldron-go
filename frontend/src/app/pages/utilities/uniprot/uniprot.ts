import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, FormControl, Validators } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { ImportedFileSelection } from '../../../components/imported-file-selection/imported-file-selection';
import { Wails } from '../../../core/services/wails';
import { getUniprotFromFields, uniprotColumns, uniprotSections } from 'uniprotparserjs';

@Component({
  selector: 'app-uniprot',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatIconModule,
    ImportedFileSelection
  ],
  templateUrl: './uniprot.html',
  styleUrl: './uniprot.scss',
})
export class Uniprot implements OnInit {
  protected form: FormGroup;
  protected running = signal(false);
  protected columns: string[] = [];

  protected uniprotFromFields: {groupName: string, items: any[]}[] = [];
  protected uniprotSections = uniprotSections;
  protected uniprotSectionMap: any = {};
  protected columnFormMap: {[key: string]: FormGroup<{columns: FormControl<string[] | null>}>} = {};

  constructor(
    private fb: FormBuilder,
    private wails: Wails
  ) {
    this.form = this.fb.group({
      inputFile: ['', Validators.required],
      from: ['UniProtKB_AC-ID', Validators.required],
      column: ['', Validators.required],
      selectedUniProtColumns: [[], Validators.required]
    });

    for (const section of uniprotColumns) {
      if (!this.uniprotSectionMap[section.section]) {
        this.uniprotSectionMap[section.section] = [];
      }
      this.uniprotSectionMap[section.section].push(section);
    }

    for (const section of uniprotSections) {
      this.columnFormMap[section] = this.fb.group({
        columns: new FormControl<string[]>([])
      });
    }
  }

  async ngOnInit() {
    try {
      const fields = await getUniprotFromFields();
      this.uniprotFromFields = fields as any;
    } catch (error) {
      console.error('Failed to load UniProt fields:', error);
    }
  }

  async openFile() {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls['inputFile'].setValue(path);
        const preview = await this.wails.parseDataFile(path, 1);
        if (preview && preview.headers) {
          this.columns = preview.headers;
        }
      }
    } catch (error) {
      console.error('Failed to open file dialog:', error);
    }
  }

  updateFormWithSelected(e: string, formControl: string) {
    this.form.controls[formControl].setValue(e);
  }

  updateColumns(cols: string[]) {
    this.columns = cols;
  }

  async submit() {
    if (this.form.invalid) return;

    let selectedColumns: string[] = [];
    for (const section of uniprotSections) {
      if (this.columnFormMap[section]) {
        const sectionColumns = this.columnFormMap[section].value.columns;
        if (sectionColumns) {
          selectedColumns.push(...sectionColumns);
        }
      }
    }

    if (selectedColumns.length === 0) {
      alert('Please select at least one UniProt field');
      return;
    }

    this.running.set(true);
    try {
      const jobId = await this.wails.createJob({
        type: 'uniprot',
        name: 'UniProt Data Retrieval',
        inputFiles: [this.form.value.inputFile],
        parameters: {
          input_file: this.form.value.inputFile,
          from: this.form.value.from,
          column: this.form.value.column,
          selected_uniprot_columns: selectedColumns
        }
      });
      console.log('UniProt job created:', jobId);
    } catch (error) {
      console.error('Failed to create UniProt job:', error);
    } finally {
      this.running.set(false);
    }
  }

  reset() {
    this.form.reset({
      inputFile: '',
      from: 'UniProtKB_AC-ID',
      column: '',
      selectedUniProtColumns: []
    });
    this.columns = [];
    for (const section of uniprotSections) {
      this.columnFormMap[section].reset({ columns: [] });
    }
  }
}
