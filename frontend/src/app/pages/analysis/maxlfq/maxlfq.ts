import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { ImportedFileSelection } from '../../../components/imported-file-selection/imported-file-selection';
import { EnvironmentIndicator } from '../../../components/environment-indicator/environment-indicator';
import { Wails } from '../../../core/services/wails';

@Component({
  selector: 'app-maxlfq',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatIconModule,
    ImportedFileSelection,
    EnvironmentIndicator
  ],
  templateUrl: './maxlfq.html',
  styleUrl: './maxlfq.scss'
})
export class Maxlfq implements OnInit {
  protected form: FormGroup;
  protected columns: string[] = [];
  protected running = signal(false);
  protected createdJobId: string | null = null;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router
  ) {
    this.form = this.fb.group({
      filePath: ['', Validators.required],
      proteinCol: ['Protein.Group', Validators.required],
      peptideCol: ['Precursor.Id', Validators.required],
      sampleCols: [[], Validators.required],
      minSamples: [1, [Validators.required, Validators.min(1)]],
      useLog2: [false],
      normalize: [true]
    });
  }

  ngOnInit() {}

  async openFile() {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls['filePath'].setValue(path);
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

    this.running.set(true);
    this.createdJobId = null;
    try {
      const jobId = await this.wails.createJob({
        type: 'maxlfq',
        name: 'MaxLFQ Normalization',
        inputFiles: [this.form.value.filePath],
        parameters: {
          protein_col: this.form.value.proteinCol,
          peptide_col: this.form.value.peptideCol,
          sample_cols: this.form.value.sampleCols,
          min_samples: this.form.value.minSamples,
          use_log2: this.form.value.useLog2,
          normalize: this.form.value.normalize
        }
      });
      console.log('MaxLFQ job created:', jobId);
      this.createdJobId = jobId;
    } catch (error) {
      console.error('Failed to create MaxLFQ job:', error);
    } finally {
      this.running.set(false);
    }
  }

  viewJob() {
    if (this.createdJobId) {
      this.router.navigate(['/job', this.createdJobId]);
    }
  }

  reset() {
    this.form.reset({
      filePath: '',
      proteinCol: 'Protein.Group',
      peptideCol: 'Precursor.Id',
      sampleCols: [],
      minSamples: 1,
      useLog2: false,
      normalize: true
    });
    this.columns = [];
    this.createdJobId = null;
  }

  async loadExample() {
    try {
      const filePath = await this.wails.getExampleFilePath('diann', 'report.pg_matrix.tsv');
      this.form.patchValue({ filePath });
      const preview = await this.wails.parseDataFile(filePath, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
        const sampleColumns = this.columns.filter(h => !['Protein.Group', 'Protein.Ids', 'Protein.Names', 'Genes', 'First.Protein.Description'].includes(h));
        this.form.patchValue({
          proteinCol: 'Protein.Group',
          peptideCol: 'Precursor.Id',
          sampleCols: sampleColumns.slice(0, 10)
        });
      }
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
