import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { EnvironmentIndicator } from '../../../components/environment-indicator/environment-indicator';
import { Wails } from '../../../core/services/wails';

@Component({
  selector: 'app-cv-plot',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatIconModule,
    EnvironmentIndicator
  ],
  templateUrl: './cv-plot.html',
  styleUrl: './cv-plot.scss',
})
export class CvPlot implements OnInit {
  protected form: FormGroup;
  protected running = false;
  protected createdJobId: string | null = null;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router
  ) {
    this.form = this.fb.group({
      logFilePath: [''],
      reportPrFilePath: ['', Validators.required],
      reportPgFilePath: ['', Validators.required],
      intensityCol: ['Intensity'],
      annotationFile: ['', Validators.required],
      sampleNames: ['']
    });
  }

  ngOnInit() {}

  async openFile(controlName: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls[controlName].setValue(path);
      }
    } catch (error) {
      await this.wails.logToFile('[CvPlot] Failed to open file: ' + error);
    }
  }

  async submit() {
    if (this.form.invalid) return;

    this.running = true;
    this.createdJobId = null;
    try {
      const inputFiles = [
        this.form.value.reportPrFilePath,
        this.form.value.reportPgFilePath,
        this.form.value.annotationFile
      ].filter(f => f);

      if (this.form.value.logFilePath) {
        inputFiles.push(this.form.value.logFilePath);
      }

      const jobId = await this.wails.createJob({
        type: 'cv-plot',
        name: 'CV Plot',
        inputFiles: inputFiles,
        parameters: {
          log_file_path: this.form.value.logFilePath || '',
          report_pr_file_path: this.form.value.reportPrFilePath,
          report_pg_file_path: this.form.value.reportPgFilePath,
          intensity_col: this.form.value.intensityCol,
          annotation_file: this.form.value.annotationFile,
          sample_names: this.form.value.sampleNames
        }
      });
      await this.wails.logToFile('[CvPlot] Job created: ' + jobId);
      this.createdJobId = jobId;
    } catch (error) {
      await this.wails.logToFile('[CvPlot] Failed to create job: ' + error);
    } finally {
      this.running = false;
    }
  }

  viewJob() {
    if (this.createdJobId) {
      this.router.navigate(['/job', this.createdJobId]);
    }
  }

  reset() {
    this.form.reset({
      logFilePath: '',
      reportPrFilePath: '',
      reportPgFilePath: '',
      intensityCol: 'Intensity',
      annotationFile: '',
      sampleNames: ''
    });
    this.createdJobId = null;
  }

  async loadExample() {
    try {
      const inputFilePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      const annotationFilePath = await this.wails.getExampleFilePath('diann', 'annotation.txt');

      this.form.patchValue({
        reportPrFilePath: inputFilePath,
        reportPgFilePath: inputFilePath,
        annotationFile: annotationFilePath
      });
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
