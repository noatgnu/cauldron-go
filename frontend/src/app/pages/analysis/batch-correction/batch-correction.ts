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
import { MatSnackBar } from '@angular/material/snack-bar';
import { ImportedFileSelection } from '../../../components/imported-file-selection/imported-file-selection';
import { EnvironmentIndicator } from '../../../components/environment-indicator/environment-indicator';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';

@Component({
  selector: 'app-batch-correction',
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
  templateUrl: './batch-correction.html',
  styleUrl: './batch-correction.scss'
})
export class BatchCorrection implements OnInit {
  protected form: FormGroup;
  protected columns: string[] = [];
  protected running = signal(false);
  protected createdJobId: string | null = null;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router,
    private notificationService: NotificationService,
    private snackBar: MatSnackBar
  ) {
    this.form = this.fb.group({
      filePath: ['', Validators.required],
      batchInfo: ['', Validators.required],
      method: ['combat', Validators.required],
      preserveGroup: [''],
      useLog2: [false]
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

  async openBatchFile() {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls['batchInfo'].setValue(path);
      }
    } catch (error) {
      console.error('Failed to open batch file dialog:', error);
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
        type: 'batch-correction',
        name: 'Batch Correction',
        inputFiles: [this.form.value.filePath],
        parameters: {
          batch_info: this.form.value.batchInfo,
          method: this.form.value.method,
          preserve_group: this.form.value.preserveGroup,
          use_log2: this.form.value.useLog2
        }
      });
      console.log('Batch correction job created:', jobId);
      this.createdJobId = jobId;
    } catch (error) {
      console.error('Failed to create batch correction job:', error);
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
      batchInfo: '',
      method: 'combat',
      preserveGroup: '',
      useLog2: false
    });
    this.columns = [];
    this.createdJobId = null;
  }

  async loadExample() {
    try {
      const filePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      const batchInfoPath = await this.wails.getExampleFilePath('differential_analysis', 'batch_info.txt');

      this.form.patchValue({
        filePath,
        batchInfo: batchInfoPath,
        method: 'combat'
      });
    } catch (error) {
      this.snackBar.open('Failed to load example data. Please ensure example files are available.', 'Close', {
        duration: 5000,
        horizontalPosition: 'center',
        verticalPosition: 'top',
        panelClass: ['error-snackbar']
      });
    }
  }
}
