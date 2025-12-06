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
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatChipsModule } from '@angular/material/chips';
import { EnvironmentIndicator } from '../../../components/environment-indicator/environment-indicator';
import { Wails } from '../../../core/services/wails';

@Component({
  selector: 'app-estimation-plot',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatIconModule,
    MatCheckboxModule,
    MatChipsModule,
    EnvironmentIndicator
  ],
  templateUrl: './estimation-plot.html',
  styleUrl: './estimation-plot.scss',
})
export class EstimationPlot implements OnInit {
  protected form: FormGroup;
  protected running = false;
  protected proteins: string[] = [];
  protected createdJobId: string | null = null;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router
  ) {
    this.form = this.fb.group({
      inputFile: ['', Validators.required],
      indexCol: ['', Validators.required],
      selectedProteins: [[], Validators.required],
      sampleAnnotation: ['', Validators.required],
      log2: [true],
      conditionOrder: ['']
    });
  }

  ngOnInit() {}

  async openFile(controlName: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls[controlName].setValue(path);
        if (controlName === 'inputFile') {
          const preview = await this.wails.parseDataFile(path, 1);
          if (preview && preview.headers) {
            this.proteins = preview.headers;
          }
        }
      }
    } catch (error) {
      await this.wails.logToFile('[EstimationPlot] Failed to open file: ' + error);
    }
  }

  toggleProtein(protein: string) {
    const selected = this.form.get('selectedProteins')?.value || [];
    const index = selected.indexOf(protein);
    if (index > -1) {
      selected.splice(index, 1);
    } else {
      selected.push(protein);
    }
    this.form.get('selectedProteins')?.setValue([...selected]);
  }

  isProteinSelected(protein: string): boolean {
    const selected = this.form.get('selectedProteins')?.value || [];
    return selected.includes(protein);
  }

  async submit() {
    if (this.form.invalid) return;

    this.running = true;
    this.createdJobId = null;
    try {
      const jobId = await this.wails.createJob({
        type: 'estimation-plot',
        name: 'Estimation Plot',
        inputFiles: [
          this.form.value.inputFile,
          this.form.value.sampleAnnotation
        ],
        parameters: {
          file_path: this.form.value.inputFile,
          index_col: this.form.value.indexCol,
          selected_protein: this.form.value.selectedProteins,
          sample_annotation: this.form.value.sampleAnnotation,
          log2: this.form.value.log2,
          condition_order: this.form.value.conditionOrder
        }
      });
      await this.wails.logToFile('[EstimationPlot] Job created: ' + jobId);
      this.createdJobId = jobId;
    } catch (error) {
      await this.wails.logToFile('[EstimationPlot] Failed to create job: ' + error);
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
      selectedProteins: [],
      sampleAnnotation: '',
      log2: true,
      conditionOrder: ''
    });
    this.proteins = [];
    this.createdJobId = null;
  }

  async loadExample() {
    try {
      const inputFilePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      const annotationFilePath = await this.wails.getExampleFilePath('differential_analysis', 'annotation.txt');

      this.form.patchValue({
        inputFile: inputFilePath,
        sampleAnnotation: annotationFilePath
      });

      const preview = await this.wails.parseDataFile(inputFilePath, 1);
      if (preview && preview.headers) {
        this.proteins = preview.headers;
      }
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }
}
