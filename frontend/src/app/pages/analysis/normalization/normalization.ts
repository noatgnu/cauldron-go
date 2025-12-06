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
  selector: 'app-normalization',
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
  templateUrl: './normalization.html',
  styleUrl: './normalization.scss'
})
export class Normalization implements OnInit {
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
      columnsName: [[], Validators.required],
      runtime: ['r', Validators.required],
      scalerType: ['robust', Validators.required],
      withCentering: [true],
      withScaling: [true],
      nQuantiles: [1000, Validators.required],
      outputDistribution: ['uniform', Validators.required],
      norm: ['l2', Validators.required],
      powerMethod: ['yeo-johnson', Validators.required]
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
        type: 'normalization',
        name: 'Normalization Analysis',
        inputFiles: [this.form.value.filePath],
        parameters: {
          runtime: this.form.value.runtime,
          columns_name: this.form.value.columnsName,
          scaler_type: this.form.value.scalerType,
          with_centering: this.form.value.withCentering,
          with_scaling: this.form.value.withScaling,
          n_quantiles: this.form.value.nQuantiles,
          output_distribution: this.form.value.outputDistribution,
          norm: this.form.value.norm,
          power_method: this.form.value.powerMethod
        }
      });
      console.log('Normalization job created:', jobId);
      this.createdJobId = jobId;
    } catch (error) {
      console.error('Failed to create normalization job:', error);
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
      columnsName: [],
      runtime: 'r',
      scalerType: 'robust',
      withCentering: true,
      withScaling: true,
      nQuantiles: 1000,
      outputDistribution: 'uniform',
      norm: 'l2',
      powerMethod: 'yeo-johnson'
    });
    this.columns = [];
    this.createdJobId = null;
  }

  async loadExample() {
    try {
      const filePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      this.form.patchValue({ filePath });
      const preview = await this.wails.parseDataFile(filePath, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
        const sampleColumns = this.columns.slice(0, 10);
        this.form.patchValue({
          columnsName: sampleColumns,
          scalerType: 'quantile'
        });
      }
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
