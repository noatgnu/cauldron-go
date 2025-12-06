import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule} from "@angular/forms";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatSelectModule} from "@angular/material/select";
import {DataFrame, fromCSV, IDataFrame} from "data-forge";
import {FuzzyClusteringPcaPlot} from "../fuzzy-clustering-pca-plot/fuzzy-clustering-pca-plot";
import {Wails} from "../../core/services/wails";

export interface FuzzyClusteringPlotModalData {
  filePathList: string[];
  explainedVariance: number[];
}

@Component({
  selector: 'app-fuzzy-clustering-plot-modal',
  imports: [
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatSelectModule,
    ReactiveFormsModule,
    FuzzyClusteringPcaPlot
  ],
  templateUrl: './fuzzy-clustering-plot-modal.html',
  styleUrl: './fuzzy-clustering-plot-modal.scss',
})
export class FuzzyClusteringPlotModal {
  filePathList: string[] = [];
  explainedVariance: number[] = [];

  form!: FormGroup;
  data: IDataFrame<number, {x: number, y: number, Sample: string, Condition: string, cluster: number}> = new DataFrame();
  revision: number = 0;

  constructor(
    public dialogRef: MatDialogRef<FuzzyClusteringPlotModal>,
    private wails: Wails,
    private fb: FormBuilder,
    @Inject(MAT_DIALOG_DATA) public modalData: FuzzyClusteringPlotModalData
  ) {
    if (modalData) {
      this.filePathList = modalData.filePathList || [];
      this.explainedVariance = modalData.explainedVariance || [];
    }

    this.form = this.fb.group({
      selectedFile: new FormControl(this.filePathList.length > 0 ? this.filePathList[0] : null)
    });

    this.form.controls["selectedFile"].valueChanges.subscribe((value) => {
      if (value) {
        this.loadFile(value);
      }
    });

    if (this.filePathList.length > 0) {
      this.loadFile(this.filePathList[0]);
    }
  }

  async loadFile(filePath: string) {
    try {
      const fileContent = await this.wails.readFile(filePath);
      this.data = fromCSV(fileContent);
      this.revision += 1;
    } catch (error) {
      console.error('Failed to load fuzzy clustering data file:', error);
    }
  }

  getFileName(path: string): string {
    return path.split('/').pop() || path.split('\\').pop() || path;
  }

  close() {
    this.dialogRef.close();
  }
}
