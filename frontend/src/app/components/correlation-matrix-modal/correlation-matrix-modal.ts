import {Component, Inject} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, FormsModule, ReactiveFormsModule, Validators} from "@angular/forms";
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {ImportedFileSelection} from "../imported-file-selection/imported-file-selection";
import {Wails} from "../../core/services/wails";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";
import {MatSelectModule} from "@angular/material/select";
import {MatCheckboxModule} from "@angular/material/checkbox";
import {MatIconModule} from "@angular/material/icon";

export interface CorrelationMatrixConfig {
  file_path: string;
  sample_cols: string[];
  index_col: string;
  method: string;
  min_value: number | null;
  order: string | null;
  hclust_method: string;
  presenting_method: string;
  cor_shape: string;
  plot_only: boolean;
  colorRamp: string;
}

@Component({
  selector: 'app-correlation-matrix-modal',
  imports: [
    ReactiveFormsModule,
    ImportedFileSelection,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    FormsModule,
    MatIconModule
  ],
  templateUrl: './correlation-matrix-modal.html',
  styleUrl: './correlation-matrix-modal.scss'
})
export class CorrelationMatrixModal {
  correlationPlotShape: string[] = ["full", "lower", "upper"];
  presentingMethod: string[] = ["circle", "square", "ellipse", "number", "shade", "color", "pie"];
  correlationMethod: string[] = ["pearson", "spearman", "kendall"];
  orderList: string[] = ["original", "AOE", "FPC", "hclust", "alphabet"];
  hclusteringMethod: string[] = ["complete", "ward", "ward.D", "ward.D2", "single", "average",
    "mcquitty", "median", "centroid"];
  colorRamp: string[] = "#053061,#2166AC,#4393C3,#92C5DE,#D1E5F0,#FFFFFF,#FDDBC7,#F4A582,#D6604D,#B2182B,#67001F".split(",");

  form!: FormGroup;
  columns: string[] = [];

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<CorrelationMatrixModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {
    this.form = this.fb.group({
      file_path: new FormControl(null),
      sample_cols: new FormControl([], Validators.required),
      index_col: new FormControl(null, Validators.required),
      method: new FormControl("pearson", Validators.required),
      min_value: new FormControl(null),
      order: new FormControl(null),
      hclust_method: new FormControl("ward.D"),
      presenting_method: new FormControl("ellipse"),
      cor_shape: new FormControl("upper"),
      plot_only: new FormControl(false),
      colorRamp: new FormControl(this.colorRamp.join(','), Validators.required),
    });
  }

  close() {
    this.dialogRef.close();
  }

  submit() {
    this.form.controls['colorRamp'].setValue(this.colorRamp.join(','));
    if (this.form.valid) {
      this.dialogRef.close(this.form.value);
    }
  }

  async openFile(fileType: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls[fileType].setValue(path);
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

  addColor() {
    this.colorRamp.push("#FFFFFF");
  }

  removeColor(index: number) {
    this.colorRamp.splice(index, 1);
  }
}
