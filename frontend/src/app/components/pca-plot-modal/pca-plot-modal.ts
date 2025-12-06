import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {MatButtonModule} from "@angular/material/button";
import {DataFrame, fromCSV, IDataFrame} from "data-forge";
import {PcaPlot} from "../pca-plot/pca-plot";
import {Wails} from "../../core/services/wails";

export interface PcaPlotModalData {
  jobId: string;
  annotation: IDataFrame<number, {Sample: string, Condition: string}>;
  txtFile: string;
  jsonFile: string;
}

@Component({
  selector: 'app-pca-plot-modal',
  imports: [
    MatDialogModule,
    MatButtonModule,
    PcaPlot
  ],
  templateUrl: './pca-plot-modal.html',
  styleUrl: './pca-plot-modal.scss',
})
export class PcaPlotModal {
  pcaData: IDataFrame<number, {x_pca: number, y_pca: number, z_pca?: number, sample: string}> = new DataFrame();
  jobId: string = '';
  annotation: IDataFrame<number, {Sample: string, Condition: string}> = new DataFrame();
  explainedVariance: number[] = [];

  constructor(
    public dialogRef: MatDialogRef<PcaPlotModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: PcaPlotModalData
  ) {
    if (data) {
      this.jobId = data.jobId || '';
      this.annotation = data.annotation || new DataFrame();

      if (data.txtFile) {
        this.loadTxtFile(data.txtFile);
      }

      if (data.jsonFile) {
        this.loadJsonFile(data.jsonFile);
      }
    }
  }

  async loadTxtFile(filePath: string) {
    try {
      const fileContent = await this.wails.readFile(filePath);
      this.pcaData = fromCSV(fileContent);
    } catch (error) {
      console.error('Failed to load PCA data file:', error);
    }
  }

  async loadJsonFile(filePath: string) {
    try {
      const fileContent = await this.wails.readFile(filePath);
      this.explainedVariance = JSON.parse(fileContent);
    } catch (error) {
      console.error('Failed to load explained variance file:', error);
    }
  }

  close() {
    this.dialogRef.close();
  }
}
