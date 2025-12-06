import {Component} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators} from "@angular/forms";
import {MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {Wails} from "../../core/services/wails";
import {ImportedFileSelection} from "../imported-file-selection/imported-file-selection";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";
import {MatSelectModule} from "@angular/material/select";
import {MatCheckboxModule} from "@angular/material/checkbox";
import {MatIconModule} from "@angular/material/icon";
import {MatAutocompleteModule} from "@angular/material/autocomplete";
import {MatListModule} from "@angular/material/list";
import {IDataFrame, DataFrame, fromCSV} from "data-forge";
import {map, startWith} from "rxjs";

export interface EstimationPlotConfig {
  input_file: string | null;
  annotation_file: string | null;
  index_col: string | null;
  log2: boolean;
  condition_order: string[];
  selected_protein: string | null;
}

@Component({
  selector: 'app-estimation-plot-modal',
  imports: [
    ReactiveFormsModule,
    ImportedFileSelection,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatIconModule,
    MatAutocompleteModule,
    MatListModule
  ],
  templateUrl: './estimation-plot-modal.html',
  styleUrl: './estimation-plot-modal.scss',
})
export class EstimationPlotModal {
  form!: FormGroup;
  columns: string[] = [];
  annotations: IDataFrame = new DataFrame();
  conditions: string[] = [];
  data: string[] = [];
  filteredData: string[] = [];

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<EstimationPlotModal>,
    private wails: Wails
  ) {
    this.form = this.fb.group({
      input_file: new FormControl<string | null>(null, Validators.required),
      annotation_file: new FormControl<string | null>(null, Validators.required),
      index_col: new FormControl<string | null>(null, Validators.required),
      log2: new FormControl<boolean>(true),
      condition_order: new FormControl<string[]>([], Validators.required),
      selected_protein: new FormControl<string | null>(null, Validators.required),
    });

    this.form.controls['index_col'].valueChanges.subscribe(async (value) => {
      if (value && this.form.controls['input_file'].value) {
        try {
          const fileContent = await this.wails.readFile(this.form.controls['input_file'].value);
          const df = fromCSV(fileContent);
          this.data = df.getSeries(value).distinct().toArray();
        } catch (error) {
          console.error('Failed to read file for index column:', error);
        }
      }
    });

    this.form.controls['selected_protein'].valueChanges.pipe(
      startWith(''),
      map(value => this._filter(value || ''))
    ).subscribe(filtered => {
      this.filteredData = filtered;
    });
  }

  private _filter(value: string): string[] {
    const filterValue = value.toLowerCase();
    return this.data.filter(option => option.toLowerCase().includes(filterValue)).slice(0, 10);
  }

  async openFile(fileType: 'input_file' | 'annotation_file') {
    try {
      const path = await this.wails.openDataFileDialog();

      if (path) {
        this.form.controls[fileType].setValue(path);

        if (fileType === 'annotation_file') {
          const fileContent = await this.wails.readFile(path);
          const df = fromCSV(fileContent);
          this.annotations = df;
          this.conditions = df.getSeries('Condition').distinct().toArray();
          this.form.controls['condition_order'].setValue(this.conditions);
        } else {
          const preview = await this.wails.parseDataFile(path, 1);
          if (preview && preview.headers) {
            this.columns = preview.headers;
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

  moveConditionUp(index: number) {
    if (index > 0) {
      const temp = this.conditions[index];
      this.conditions[index] = this.conditions[index - 1];
      this.conditions[index - 1] = temp;
      this.form.controls['condition_order'].setValue([...this.conditions]);
    }
  }

  moveConditionDown(index: number) {
    if (index < this.conditions.length - 1) {
      const temp = this.conditions[index];
      this.conditions[index] = this.conditions[index + 1];
      this.conditions[index + 1] = temp;
      this.form.controls['condition_order'].setValue([...this.conditions]);
    }
  }

  async loadExample() {
    try {
      const inputFilePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      const annotationFilePath = await this.wails.getExampleFilePath('differential_analysis', 'annotation.txt');

      this.form.controls['input_file'].setValue(inputFilePath);
      const preview = await this.wails.parseDataFile(inputFilePath, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
      }

      this.form.controls['annotation_file'].setValue(annotationFilePath);
      const annotationContent = await this.wails.readFile(annotationFilePath);
      const annotationDf = fromCSV(annotationContent);
      this.annotations = annotationDf;
      this.conditions = annotationDf.getSeries('Condition').distinct().toArray();
      this.form.controls['condition_order'].setValue(this.conditions);

      this.form.controls['log2'].setValue(true);
      this.form.controls['index_col'].setValue('Precursor.Id');
    } catch (error) {
      console.error('Failed to load example:', error);
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }

  close() {
    this.dialogRef.close();
  }

  submit() {
    if (this.form.valid) {
      this.dialogRef.close(this.form.value);
    }
  }
}
