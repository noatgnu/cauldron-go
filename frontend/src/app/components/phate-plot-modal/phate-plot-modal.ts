import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {MatButtonModule} from "@angular/material/button";
import {DataFrame, fromCSV, IDataFrame} from "data-forge";
import {PhatePlot} from "../phate-plot/phate-plot";
import {Wails} from "../../core/services/wails";

export interface PhatePlotModalData {
  jobId: string;
  annotation: IDataFrame<number, {Sample: string, Condition: string}>;
  txtFile: string;
}

@Component({
  selector: 'app-phate-plot-modal',
  imports: [
    MatDialogModule,
    MatButtonModule,
    PhatePlot
  ],
  templateUrl: './phate-plot-modal.html',
  styleUrl: './phate-plot-modal.scss',
})
export class PhatePlotModal {
  phateData: IDataFrame<number, {x_phate: number, y_phate: number, z_phate?: number, sample: string}> = new DataFrame();
  jobId: string = '';
  annotation: IDataFrame<number, {Sample: string, Condition: string}> = new DataFrame();

  constructor(
    public dialogRef: MatDialogRef<PhatePlotModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: PhatePlotModalData
  ) {
    if (data) {
      this.jobId = data.jobId || '';
      this.annotation = data.annotation || new DataFrame();

      if (data.txtFile) {
        this.loadTxtFile(data.txtFile);
      }
    }
  }

  async loadTxtFile(filePath: string) {
    try {
      const fileContent = await this.wails.readFile(filePath);
      this.phateData = fromCSV(fileContent);
    } catch (error) {
      console.error('Failed to load PHATE data file:', error);
    }
  }

  close() {
    this.dialogRef.close();
  }
}
