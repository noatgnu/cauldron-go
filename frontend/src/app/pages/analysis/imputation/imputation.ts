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
  selector: 'app-imputation',
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
  templateUrl: './imputation.html',
  styleUrl: './imputation.scss'
})
export class Imputation implements OnInit {
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
      runtime: ['r', Validators.required],
      columns: [[], Validators.required],
      method: ['knn', Validators.required],
      k: [5, [Validators.required, Validators.min(1)]],
      strategy: ['mean', Validators.required],
      fillValue: [0, Validators.required],
      iterations: [10, [Validators.required, Validators.min(1)]]
    });
  }

  ngOnInit() {
    this.form.get('method')?.valueChanges.subscribe(() => {
      this.updateValidators();
    });

    this.form.get('strategy')?.valueChanges.subscribe(() => {
      this.updateValidators();
    });
  }

  private updateValidators() {
    const method = this.form.get('method')?.value;
    const strategy = this.form.get('strategy')?.value;

    this.form.get('k')?.clearValidators();
    this.form.get('strategy')?.clearValidators();
    this.form.get('fillValue')?.clearValidators();
    this.form.get('iterations')?.clearValidators();

    if (method === 'knn') {
      this.form.get('k')?.setValidators([Validators.required, Validators.min(1)]);
    } else if (method === 'simple') {
      this.form.get('strategy')?.setValidators([Validators.required]);
      if (strategy === 'constant') {
        this.form.get('fillValue')?.setValidators([Validators.required]);
      }
    } else if (method === 'iterative') {
      this.form.get('iterations')?.setValidators([Validators.required, Validators.min(1)]);
    }

    this.form.get('k')?.updateValueAndValidity();
    this.form.get('strategy')?.updateValueAndValidity();
    this.form.get('fillValue')?.updateValueAndValidity();
    this.form.get('iterations')?.updateValueAndValidity();
  }

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
      const formValue = this.form.value;
      const parameters: any = {
        runtime: formValue.runtime,
        columns: formValue.columns,
        method: formValue.method
      };

      if (formValue.method === 'knn') {
        parameters.k = formValue.k;
      } else if (formValue.method === 'simple') {
        parameters.strategy = formValue.strategy;
        if (formValue.strategy === 'constant') {
          parameters.fillValue = formValue.fillValue;
        }
      } else if (formValue.method === 'iterative') {
        parameters.iterations = formValue.iterations;
      }

      const jobId = await this.wails.createJob({
        type: 'imputation',
        name: 'Imputation Analysis',
        inputFiles: [formValue.filePath],
        parameters
      });
      console.log('Imputation job created:', jobId);
      this.createdJobId = jobId;
    } catch (error) {
      console.error('Failed to create imputation job:', error);
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
      runtime: 'r',
      columns: [],
      method: 'knn',
      k: 5,
      strategy: 'mean',
      fillValue: 0,
      iterations: 10
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
        const sampleColumns = preview.headers.filter(h => !['Protein.Ids', 'Precursor.Id', 'Genes'].includes(h));
        this.form.patchValue({
          columns: sampleColumns.slice(0, 10),
          method: 'knn',
          k: 5
        });
      }
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
