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
  selector: 'app-venn-diagram',
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
  templateUrl: './venn-diagram.html',
  styleUrl: './venn-diagram.scss'
})
export class VennDiagram implements OnInit {
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
      sampleCols: [[], Validators.required],
      setNames: ['', Validators.required],
      threshold: [0, Validators.required],
      usePresence: [true],
      fillColors: [''],
      alpha: [0.5, [Validators.required, Validators.min(0), Validators.max(1)]]
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
        type: 'venn-diagram',
        name: 'Venn Diagram',
        inputFiles: [this.form.value.filePath],
        parameters: {
          sample_cols: this.form.value.sampleCols,
          set_names: this.form.value.setNames,
          threshold: this.form.value.threshold,
          use_presence: this.form.value.usePresence,
          fill_colors: this.form.value.fillColors,
          alpha: this.form.value.alpha
        }
      });
      console.log('Venn diagram job created:', jobId);
      this.createdJobId = jobId;
    } catch (error) {
      console.error('Failed to create venn diagram job:', error);
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
      sampleCols: [],
      setNames: '',
      threshold: 0,
      usePresence: true,
      fillColors: '',
      alpha: 0.5
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
        const sampleColumns = this.columns.filter(h => !['Protein.Ids', 'Precursor.Id', 'Genes'].includes(h));
        const selectedCols = sampleColumns.slice(0, 3);
        this.form.patchValue({
          sampleCols: selectedCols,
          setNames: selectedCols.join(','),
          threshold: 0,
          usePresence: true
        });
      }
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
