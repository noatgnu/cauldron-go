import { Component, OnInit } from '@angular/core';
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
import { NotificationService } from '../../../core/services/notification.service';
import { fromCSV } from 'data-forge';

@Component({
  selector: 'app-pca',
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
  templateUrl: './pca.html',
  styleUrl: './pca.scss'
})
export class Pca implements OnInit {
  protected form: FormGroup;
  protected columns: string[] = [];
  protected running = false;
  protected createdJobId: string | null = null;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private router: Router,
    private notificationService: NotificationService
  ) {
    this.form = this.fb.group({
      inputFile: ['', Validators.required],
      columnsName: [[], Validators.required],
      nComponents: [2, [Validators.required, Validators.min(2), Validators.max(3)]],
      log2: [false]
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
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.wails.logToFile(`[PCA] Failed to open file dialog: ${errorMsg}`);
      this.notificationService.showError(`Failed to open file: ${errorMsg}`);
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
      await this.wails.logToFile('[PCA] Starting job creation...');
      const jobId = await this.wails.createJob({
        type: 'pca',
        name: 'PCA Analysis',
        inputFiles: [this.form.value.inputFile],
        parameters: {
          columns_name: this.form.value.columnsName,
          n_components: this.form.value.nComponents,
          log2: this.form.value.log2
        }
      });
      await this.wails.logToFile(`[PCA] Job created: ${jobId}`);
      this.createdJobId = jobId;
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      try {
        await this.wails.logToFile(`[PCA] Failed to create job: ${errorMsg}`);
      } catch {}
      this.notificationService.showError(`Failed to create PCA job: ${errorMsg}`);
    } finally {
      this.running = false;
      try {
        await this.wails.logToFile(`[PCA] Job creation completed, running=${this.running}, jobId=${this.createdJobId}`);
      } catch {}
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
      columnsName: [],
      nComponents: 2,
      log2: false
    });
    this.columns = [];
    this.createdJobId = null;
  }

  async loadExample() {
    try {
      const inputFilePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      const phateFilePath = await this.wails.getExampleFilePath('phate', 'phate_output.txt');

      this.form.patchValue({ inputFile: inputFilePath });

      const preview = await this.wails.parseDataFile(inputFilePath, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
      }

      const phateContent = await this.wails.readFile(phateFilePath);
      // @ts-ignore - fromCSV doesn't properly type its delimiter option
      const phateDf = fromCSV(phateContent, { delimiter: '\t' });
      const sampleValues = phateDf.getSeries('sample').toArray();

      this.form.patchValue({
        columnsName: sampleValues,
        nComponents: 2,
        log2: true
      });
    } catch (error) {
      this.notificationService.showError('Failed to load example data. Please ensure example files are available.');
    }
  }
}
