import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators, FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { ImportedFileSelection } from '../../../components/imported-file-selection/imported-file-selection';
import { EnvironmentIndicator } from '../../../components/environment-indicator/environment-indicator';
import { Wails } from '../../../core/services/wails';
import { fromCSV } from 'data-forge';

interface Comparison {
  condition_A: string;
  condition_B: string;
  comparison_label: string;
}

@Component({
  selector: 'app-limma',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    FormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatIconModule,
    MatTableModule,
    ImportedFileSelection,
    EnvironmentIndicator
  ],
  templateUrl: './limma.html',
  styleUrl: './limma.scss'
})
export class Limma implements OnInit {
  protected form: FormGroup;
  protected columns: string[] = [];
  protected conditions = signal<string[]>([]);
  protected comparisons = signal<Comparison[]>([]);
  protected running = signal(false);
  protected displayedColumns = ['conditionA', 'conditionB', 'label', 'actions'];
  protected createdJobId: string | null = null;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router
  ) {
    this.form = this.fb.group({
      inputFile: ['', Validators.required],
      annotationFile: ['', Validators.required],
      indexCol: [[], Validators.required],
      log2: [false]
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
        } else if (fileType === 'annotationFile') {
          const preview = await this.wails.parseDataFile(path, 100);
          if (preview.headers.includes('Condition')) {
            const conditionIndex = preview.headers.indexOf('Condition');
            const uniqueConditions = [...new Set(preview.rows.map(row => row[conditionIndex]))];
            this.conditions.set(uniqueConditions);
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

  addComparison() {
    this.comparisons.update(comps => [
      ...comps,
      { condition_A: '', condition_B: '', comparison_label: '' }
    ]);
  }

  removeComparison(index: number) {
    this.comparisons.update(comps => comps.filter((_, i) => i !== index));
  }

  async submit() {
    if (this.form.invalid || this.comparisons().length === 0) return;

    const comparisons = this.comparisons();
    const conditions = this.conditions();

    for (const c of comparisons) {
      if (!conditions.includes(c.condition_A) || !conditions.includes(c.condition_B) || !c.comparison_label) {
        alert('Please select valid conditions for all comparisons');
        return;
      }
    }

    this.running.set(true);
    this.createdJobId = null;
    try {
      const jobId = await this.wails.createJob({
        type: 'limma',
        name: 'Limma Differential Analysis',
        inputFiles: [this.form.value.inputFile, this.form.value.annotationFile],
        parameters: {
          index_col: this.form.value.indexCol,
          log2: this.form.value.log2,
          comparisons: comparisons
        }
      });
      console.log('Limma job created:', jobId);
      this.createdJobId = jobId;
    } catch (error) {
      console.error('Failed to create Limma job:', error);
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
      inputFile: '',
      annotationFile: '',
      indexCol: [],
      log2: false
    });
    this.columns = [];
    this.conditions.set([]);
    this.comparisons.set([]);
    this.createdJobId = null;
  }

  async loadExample() {
    try {
      const inputFilePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      const annotationFilePath = await this.wails.getExampleFilePath('differential_analysis', 'annotation.txt');
      const comparisonFilePath = await this.wails.getExampleFilePath('differential_analysis', 'comparison.bca.txt');

      this.form.patchValue({ inputFile: inputFilePath });

      const inputPreview = await this.wails.parseDataFile(inputFilePath, 1);
      if (inputPreview && inputPreview.headers) {
        this.columns = inputPreview.headers;
      }

      this.form.patchValue({ annotationFile: annotationFilePath });

      const annotationPreview = await this.wails.parseDataFile(annotationFilePath, 100);
      if (annotationPreview.headers.includes('Condition')) {
        const conditionIndex = annotationPreview.headers.indexOf('Condition');
        const uniqueConditions = [...new Set(annotationPreview.rows.map(row => row[conditionIndex]))];
        this.conditions.set(uniqueConditions);
      }

      const comparisonContent = await this.wails.readFile(comparisonFilePath);
      // @ts-ignore - fromCSV doesn't properly type its delimiter option
      const comparisonDf = fromCSV(comparisonContent, { delimiter: '\t' });
      const newComparisons: Comparison[] = [];
      for (const row of comparisonDf) {
        newComparisons.push({
          condition_A: row['condition_A'],
          condition_B: row['condition_B'],
          comparison_label: row['comparison_label']
        });
      }
      this.comparisons.set(newComparisons);

      this.form.patchValue({
        indexCol: ['Protein.Ids', 'Precursor.Id'],
        log2: true
      });
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
