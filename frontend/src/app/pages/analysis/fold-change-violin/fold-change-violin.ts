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
  selector: 'app-fold-change-violin',
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
  templateUrl: './fold-change-violin.html',
  styleUrl: './fold-change-violin.scss',
})
export class FoldChangeViolin implements OnInit {
  protected form: FormGroup;
  protected running = false;
  protected createdJobId: string | null = null;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router
  ) {
    this.form = this.fb.group({
      inputFile: ['', Validators.required],
      columnsPrefix: ['Difference'],
      categories: ['', Validators.required],
      matchValue: ['+'],
      foldEnrichmentCol: ['Fold enrichment'],
      organelleCol: ['Organelle'],
      comparisonCol: [''],
      colors: [''],
      figsize: ['6,10']
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
      await this.wails.logToFile('[FoldChangeViolin] Failed to open file: ' + error);
    }
  }

  async submit() {
    if (this.form.invalid) return;

    this.running = true;
    this.createdJobId = null;
    try {
      const categoriesList = this.form.value.categories
        .split(',')
        .map((c: string) => c.trim())
        .filter((c: string) => c);

      const colorsDict: Record<string, string> = {};
      if (this.form.value.colors) {
        this.form.value.colors.split(',').forEach((color: string) => {
          const [key, value] = color.split(':').map((s: string) => s.trim());
          if (key && value) {
            colorsDict[key] = value;
          }
        });
      }

      const jobId = await this.wails.createJob({
        type: 'fold-change-violin',
        name: 'Fold Change Violin Plot',
        inputFiles: [this.form.value.inputFile],
        parameters: {
          file_path: this.form.value.inputFile,
          columns_prefix: this.form.value.columnsPrefix,
          categories: categoriesList.join(','),
          match_value: this.form.value.matchValue,
          fold_enrichment_col: this.form.value.foldEnrichmentCol,
          organelle_col: this.form.value.organelleCol,
          comparison_col: this.form.value.comparisonCol || '',
          colors: colorsDict,
          figsize: this.form.value.figsize
        }
      });
      await this.wails.logToFile('[FoldChangeViolin] Job created: ' + jobId);
      this.createdJobId = jobId;
    } catch (error) {
      await this.wails.logToFile('[FoldChangeViolin] Failed to create job: ' + error);
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
      columnsPrefix: 'Difference',
      categories: '',
      matchValue: '+',
      foldEnrichmentCol: 'Fold enrichment',
      organelleCol: 'Organelle',
      comparisonCol: '',
      colors: '',
      figsize: '6,10'
    });
    this.createdJobId = null;
  }
}
