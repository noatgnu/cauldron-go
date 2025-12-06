import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { ImportedFileSelection } from '../../../components/imported-file-selection/imported-file-selection';
import { Wails } from '../../../core/services/wails';

@Component({
  selector: 'app-ptm-remap',
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
  templateUrl: './ptm-remap.html',
  styleUrl: './ptm-remap.scss',
})
export class PtmRemap implements OnInit {
  protected form: FormGroup;
  protected running = signal(false);
  protected columns: string[] = [];

  constructor(
    private fb: FormBuilder,
    private wails: Wails
  ) {
    this.form = this.fb.group({
      filePath: ['', Validators.required],
      fastaFile: [''],
      peptideColumn: ['', Validators.required],
      positionInPeptideColumn: ['', Validators.required],
      uniprotAccColumn: ['', Validators.required]
    });
  }

  ngOnInit() {}

  async openFile(fileType: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls[fileType].setValue(path);
        if (fileType === 'filePath') {
          const preview = await this.wails.parseDataFile(path, 1);
          if (preview && preview.headers) {
            this.columns = preview.headers;
          }
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

    this.running.set(true);
    try {
      const jobId = await this.wails.createJob({
        type: 'remap-ptm',
        name: 'PTM Position Remapping',
        inputFiles: [this.form.value.filePath, this.form.value.fastaFile].filter(Boolean),
        parameters: {
          file_path: this.form.value.filePath,
          fasta_file: this.form.value.fastaFile || null,
          peptide_column: this.form.value.peptideColumn,
          position_in_peptide_column: this.form.value.positionInPeptideColumn,
          uniprot_acc_column: this.form.value.uniprotAccColumn
        }
      });
      console.log('PTM Remap job created:', jobId);
    } catch (error) {
      console.error('Failed to create PTM Remap job:', error);
    } finally {
      this.running.set(false);
    }
  }

  reset() {
    this.form.reset({
      filePath: '',
      fastaFile: '',
      peptideColumn: '',
      positionInPeptideColumn: '',
      uniprotAccColumn: ''
    });
    this.columns = [];
  }

  async loadExample() {
    try {
      const filePath = await this.wails.getExampleFilePath('ptm_positions', 'different.txt');
      this.form.patchValue({ filePath });
      const preview = await this.wails.parseDataFile(filePath, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
      }
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
