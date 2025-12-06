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
import { IDataFrame, DataFrame, fromCSV } from 'data-forge';

@Component({
  selector: 'app-alphastats',
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
  templateUrl: './alphastats.html',
  styleUrl: './alphastats.scss'
})
export class Alphastats implements OnInit {
  protected form: FormGroup;
  protected columns: string[] = [];
  protected conditions = signal<string[]>([]);
  protected comparisons = signal<{condition_A: string, condition_B: string, comparison_label: string}[]>([]);
  protected running = signal(false);
  protected displayedColumns = ['conditionA', 'conditionB', 'label'];
  protected createdJobId: string | null = null;

  protected normalizationMethods = ['zscore', 'quantile', 'linear', 'vst'];
  protected imputationMethods = ['mean', 'median', 'knn', 'randomforest'];
  protected testMethods = ['wald', 'welch-ttest', 'ttest', 'sam', 'paired-ttest', 'limma'];
  protected engines = ['spectronaut', 'maxquant', 'fragpipe', 'diann', 'generic'];

  private annotations: IDataFrame = new DataFrame();

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router
  ) {
    this.form = this.fb.group({
      inputFile: ['', Validators.required],
      annotationFile: ['', Validators.required],
      engine: ['generic', Validators.required],
      evidenceFile: [''],
      indexCol: ['', Validators.required],
      dataCompleteness: [0.3, Validators.required],
      mergeColumns: [[]],
      method: ['welch-ttest', Validators.required],
      imputation: ['knn', Validators.required],
      normalization: ['quantile', Validators.required],
      log2: [true],
      batchCorrection: [false]
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
          await this.updateConditions(path);
        }
      }
    } catch (error) {
      console.error('Failed to open file dialog:', error);
    }
  }

  async updateFormWithSelected(e: string, formControl: string) {
    this.form.controls[formControl].setValue(e);
    if (formControl === 'annotationFile') {
      await this.updateConditions(e);
    }
  }

  updateColumns(cols: string[]) {
    this.columns = cols;
  }

  async updateConditions(annotationPath: string) {
    try {
      const fileContent = await this.wails.readFile(annotationPath);
      const df = fromCSV(fileContent);
      this.annotations = df;
      this.conditions.set(df.getSeries('Condition').distinct().toArray());
    } catch (error) {
      console.error('Failed to parse annotation file:', error);
    }
  }

  addContrast() {
    const current = this.comparisons();
    this.comparisons.set([...current, {condition_A: '', condition_B: '', comparison_label: ''}]);
  }

  removeContrast(index: number) {
    const current = this.comparisons();
    current.splice(index, 1);
    this.comparisons.set([...current]);
  }

  async loadExample() {
    try {
      const inputFilePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      const annotationFilePath = await this.wails.getExampleFilePath('differential_analysis', 'annotation.txt');
      const comparisonFilePath = await this.wails.getExampleFilePath('differential_analysis', 'comparison.bca.txt');

      this.form.patchValue({ inputFile: inputFilePath });
      const preview = await this.wails.parseDataFile(inputFilePath, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
      }

      this.form.patchValue({ annotationFile: annotationFilePath });
      const annotationContent = await this.wails.readFile(annotationFilePath);
      const annotationDf = fromCSV(annotationContent);
      this.annotations = annotationDf;
      this.conditions.set(annotationDf.getSeries('Condition').distinct().toArray());

      const comparisonContent = await this.wails.readFile(comparisonFilePath);
      const comparisonDf = fromCSV(comparisonContent);
      const loadedComparisons: {condition_A: string, condition_B: string, comparison_label: string}[] = [];
      for (const row of comparisonDf) {
        loadedComparisons.push({
          condition_A: row['condition_A'] as string,
          condition_B: row['condition_B'] as string,
          comparison_label: row['comparison_label'] as string
        });
      }
      this.comparisons.set(loadedComparisons);

      this.form.patchValue({
        indexCol: 'Precursor.Id',
        mergeColumns: ['Protein.Ids', 'Genes'],
        evidenceFile: '',
        dataCompleteness: 0.3,
        imputation: 'knn',
        normalization: 'quantile',
        method: 'welch-ttest',
        engine: 'generic',
        log2: true
      });
    } catch (error) {
      console.error('Failed to load example:', error);
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }

  async submit() {
    if (this.form.invalid) return;

    const comparisons = this.comparisons();
    for (const c of comparisons) {
      if (!c.condition_A || !c.condition_B || !c.comparison_label) {
        alert('Please complete all comparison fields');
        return;
      }
    }

    this.running.set(true);
    this.createdJobId = null;
    try {
      const jobId = await this.wails.createJob({
        type: 'alphastats',
        name: 'AlphaStats Differential Analysis',
        inputFiles: [this.form.value.inputFile, this.form.value.annotationFile],
        parameters: {
          input_file: this.form.value.inputFile,
          annotation_file: this.form.value.annotationFile,
          engine: this.form.value.engine,
          evidence_file: this.form.value.evidenceFile,
          index_col: this.form.value.indexCol,
          data_completeness: this.form.value.dataCompleteness,
          merge_columns_list: this.form.value.mergeColumns,
          method: this.form.value.method,
          imputation: this.form.value.imputation,
          normalization: this.form.value.normalization,
          log2: this.form.value.log2,
          batch_correction: this.form.value.batchCorrection,
          comparisons: comparisons
        }
      });
      console.log('AlphaStats job created:', jobId);
      this.createdJobId = jobId;
    } catch (error) {
      console.error('Failed to create AlphaStats job:', error);
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
      engine: 'generic',
      evidenceFile: '',
      indexCol: '',
      dataCompleteness: 0.3,
      mergeColumns: [],
      method: 'welch-ttest',
      imputation: 'knn',
      normalization: 'quantile',
      log2: true,
      batchCorrection: false
    });
    this.columns = [];
    this.conditions.set([]);
    this.comparisons.set([]);
    this.createdJobId = null;
  }
}
