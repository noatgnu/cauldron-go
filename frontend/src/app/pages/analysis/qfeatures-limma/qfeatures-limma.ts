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
  selector: 'app-qfeatures-limma',
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
  templateUrl: './qfeatures-limma.html',
  styleUrl: './qfeatures-limma.scss',
})
export class QfeaturesLimma implements OnInit {
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
      inputFile: ['', Validators.required],
      annotationFile: ['', Validators.required],
      indexCol: ['', Validators.required],
      log2: [false],
      pvalCutoff: [0.05, [Validators.required, Validators.min(0), Validators.max(1)]],
      fcCutoff: [1.5, [Validators.required, Validators.min(0)]]
    });
  }

  ngOnInit() {}

  async openFile(fileType: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls[fileType].setValue(path);
        if (fileType === 'inputFile') {
          const preview = await this.wails.parseDataFile(path, 1);
          if (preview && preview.headers) {
            this.columns = preview.headers;
          }
        }
      }
    } catch (error) {
      await this.wails.logToFile(`[QFeatures Limma] Failed to open file dialog: ${error}`);
    }
  }

  updateFormWithSelected(e: string, formControl: string) {
    this.form.controls[formControl].setValue(e);
  }

  updateColumns(cols: string[]) {
    this.columns = cols;
  }

  async loadExample() {
    try {
      const exampleInput = await this.wails.getExampleFilePath('qfeatures_limma', 'protein_data.csv');
      const exampleAnnotation = await this.wails.getExampleFilePath('qfeatures_limma', 'annotation.csv');

      this.form.patchValue({
        inputFile: exampleInput,
        annotationFile: exampleAnnotation,
        indexCol: 'Protein',
        log2: true,
        pvalCutoff: 0.05,
        fcCutoff: 1.5
      });

      const preview = await this.wails.parseDataFile(exampleInput, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
      }
    } catch (error) {
      await this.wails.logToFile(`[QFeatures Limma] Failed to load example: ${error}`);
    }
  }

  reset() {
    this.form.reset({
      inputFile: '',
      annotationFile: '',
      indexCol: '',
      log2: false,
      pvalCutoff: 0.05,
      fcCutoff: 1.5
    });
    this.columns = [];
    this.createdJobId = null;
  }

  async submit() {
    if (this.form.invalid) {
      return;
    }

    this.running.set(true);
    this.createdJobId = null;

    try {
      const jobId = await this.wails.createJob({
        type: 'qfeatures_limma',
        name: 'QFeatures + Limma Analysis',
        inputFiles: [this.form.value.inputFile, this.form.value.annotationFile],
        parameters: {
          input_file: this.form.value.inputFile,
          annotation_file: this.form.value.annotationFile,
          index_col: this.form.value.indexCol,
          log2: this.form.value.log2,
          pval_cutoff: this.form.value.pvalCutoff,
          fc_cutoff: this.form.value.fcCutoff
        }
      });

      this.createdJobId = jobId;
      await this.wails.logToFile(`[QFeatures Limma] Job created successfully: ${jobId}`);
    } catch (error: any) {
      await this.wails.logToFile(`[QFeatures Limma] Failed to create job: ${error.message || error}`);
    } finally {
      this.running.set(false);
    }
  }

  viewJob() {
    if (this.createdJobId) {
      this.router.navigate(['/job', this.createdJobId]);
    }
  }
}
