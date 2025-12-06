import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { ImportedFileSelection } from '../../../components/imported-file-selection/imported-file-selection';
import { EnvironmentIndicator } from '../../../components/environment-indicator/environment-indicator';
import { Wails } from '../../../core/services/wails';

@Component({
  selector: 'app-fuzzy-clustering',
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
    ImportedFileSelection,
    EnvironmentIndicator
  ],
  templateUrl: './fuzzy-clustering.html',
  styleUrl: './fuzzy-clustering.scss'
})
export class FuzzyClustering implements OnInit {
  protected form: FormGroup;
  protected running = signal(false);
  protected clusterOptions = [2, 3, 4, 5, 6, 7, 8, 9, 10];
  protected createdJobId: string | null = null;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router
  ) {
    this.form = this.fb.group({
      filePath: ['', Validators.required],
      annotationPath: ['', Validators.required],
      centerCount: [[3], Validators.required]
    });
  }

  ngOnInit() {}

  async openFile(fileType: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.patchValue({[fileType]: path});
      }
    } catch (error) {
      console.error('Failed to open file dialog:', error);
    }
  }

  updateFormWithSelected(event: string, fileType: string) {
    this.form.controls[fileType].setValue(event);
  }

  async loadExample() {
    try {
      const dataFilePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      const annotationFilePath = await this.wails.getExampleFilePath('differential_analysis', 'annotation.txt');

      this.form.patchValue({
        filePath: dataFilePath,
        annotationPath: annotationFilePath,
        centerCount: [3, 4, 5]
      });
    } catch (error) {
      console.error('Failed to load example:', error);
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }

  async submit() {
    if (this.form.invalid) return;

    this.running.set(true);
    this.createdJobId = null;
    try {
      const jobId = await this.wails.createJob({
        type: 'fuzzy-clustering',
        name: 'Fuzzy C-Means Clustering',
        inputFiles: [this.form.value.filePath, this.form.value.annotationPath],
        parameters: {
          file_path: this.form.value.filePath,
          annotation_path: this.form.value.annotationPath,
          center_count: this.form.value.centerCount
        }
      });
      console.log('Fuzzy Clustering job created:', jobId);
      this.createdJobId = jobId;
    } catch (error) {
      console.error('Failed to create Fuzzy Clustering job:', error);
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
      annotationPath: '',
      centerCount: [3]
    });
    this.createdJobId = null;
  }
}
