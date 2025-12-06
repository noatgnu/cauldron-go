import { Component, OnInit } from '@angular/core';
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
import { MatExpansionModule } from '@angular/material/expansion';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { ImportedFileSelection } from '../../../components/imported-file-selection/imported-file-selection';
import { EnvironmentIndicator } from '../../../components/environment-indicator/environment-indicator';
import { Wails } from '../../../core/services/wails';

@Component({
  selector: 'app-correlation-matrix',
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
    MatExpansionModule,
    MatCheckboxModule,
    ImportedFileSelection,
    EnvironmentIndicator
  ],
  templateUrl: './correlation-matrix.html',
  styleUrl: './correlation-matrix.scss'
})
export class CorrelationMatrix implements OnInit {
  protected form: FormGroup;
  protected columns: string[] = [];
  protected running = false;
  protected createdJobId: string | null = null;

  readonly methodOptions = ['pearson', 'spearman', 'kendall'];
  readonly orderOptions = ['original', 'AOE', 'FPC', 'hclust'];
  readonly hclustMethods = ['ward.D', 'ward.D2', 'single', 'complete', 'average', 'mcquitty', 'median', 'centroid'];
  readonly presentingMethods = ['circle', 'square', 'ellipse', 'number', 'pie', 'shade', 'color'];
  readonly shapeOptions = ['full', 'upper', 'lower'];

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router
  ) {
    this.form = this.fb.group({
      inputFile: ['', Validators.required],
      indexCol: ['', Validators.required],
      sampleCols: [[], Validators.required],
      method: ['pearson'],
      minValue: [null],
      order: ['hclust'],
      hclustMethod: ['ward.D'],
      presentingMethod: ['ellipse'],
      corShape: ['upper'],
      colorRampPalette: ['#053061,#2166AC,#4393C3,#92C5DE,#D1E5F0,#FFFFFF,#FDDBC7,#F4A582,#D6604D,#B2182B,#67001F'],
      plotWidth: [10],
      plotHeight: [10],
      textLabelSize: [1],
      numberLabelSize: [1],
      labelRotation: [45],
      showDiagonal: [true],
      addGrid: [false],
      gridColor: ['white'],
      numberDigits: [2],
      plotTitle: ['']
    });
  }

  ngOnInit() {}

  async openFile() {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls['inputFile'].setValue(path);
        const preview = await this.wails.parseDataFile(path, 1);
        if (preview && preview.headers) {
          this.columns = preview.headers;
        }
      }
    } catch (error) {
      await this.wails.logToFile('[CorrelationMatrix] Failed to open file: ' + error);
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

    this.running = true;
    this.createdJobId = null;
    try {
      const jobId = await this.wails.createJob({
        type: 'correlation-matrix',
        name: 'Correlation Matrix',
        inputFiles: [this.form.value.inputFile],
        parameters: {
          index_col: this.form.value.indexCol,
          sample_cols: this.form.value.sampleCols,
          method: this.form.value.method,
          min_value: this.form.value.minValue,
          order: this.form.value.order,
          hclust_method: this.form.value.hclustMethod,
          presenting_method: this.form.value.presentingMethod,
          cor_shape: this.form.value.corShape,
          color_ramp_palette: this.form.value.colorRampPalette,
          plot_width: this.form.value.plotWidth,
          plot_height: this.form.value.plotHeight,
          text_label_size: this.form.value.textLabelSize,
          number_label_size: this.form.value.numberLabelSize,
          label_rotation: this.form.value.labelRotation,
          show_diagonal: this.form.value.showDiagonal,
          add_grid: this.form.value.addGrid,
          grid_color: this.form.value.gridColor,
          number_digits: this.form.value.numberDigits,
          plot_title: this.form.value.plotTitle
        }
      });
      await this.wails.logToFile('[CorrelationMatrix] Job created: ' + jobId);
      this.createdJobId = jobId;
    } catch (error) {
      await this.wails.logToFile('[CorrelationMatrix] Failed to create job: ' + error);
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
      inputFile: '',
      indexCol: '',
      sampleCols: [],
      method: 'pearson',
      minValue: null,
      order: 'hclust',
      hclustMethod: 'ward.D',
      presentingMethod: 'ellipse',
      corShape: 'upper',
      colorRampPalette: '#053061,#2166AC,#4393C3,#92C5DE,#D1E5F0,#FFFFFF,#FDDBC7,#F4A582,#D6604D,#B2182B,#67001F',
      plotWidth: 10,
      plotHeight: 10,
      textLabelSize: 1,
      numberLabelSize: 1,
      labelRotation: 45,
      showDiagonal: true,
      addGrid: false,
      gridColor: 'white',
      numberDigits: 2,
      plotTitle: ''
    });
    this.columns = [];
    this.createdJobId = null;
  }

  async loadExample() {
    try {
      const filePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      this.form.patchValue({ inputFile: filePath });
      const preview = await this.wails.parseDataFile(filePath, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
        const sampleColumns = this.columns.slice(0, 10);
        this.form.patchValue({ sampleCols: sampleColumns });
      }
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
