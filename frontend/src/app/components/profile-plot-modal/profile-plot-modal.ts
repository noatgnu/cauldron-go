import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {MatButtonModule} from "@angular/material/button";
import {DataFrame, IDataFrame} from "data-forge";
import {ProfilePlot} from "../profile-plot/profile-plot";

export interface ProfilePlotModalData {
  annotation: IDataFrame<number, {Sample: string, Condition: string}>;
}

@Component({
  selector: 'app-profile-plot-modal',
  imports: [
    MatDialogModule,
    MatButtonModule,
    ProfilePlot
  ],
  templateUrl: './profile-plot-modal.html',
  styleUrl: './profile-plot-modal.scss',
})
export class ProfilePlotModal {
  annotation: IDataFrame<number, {Sample: string, Condition: string}> = new DataFrame();

  constructor(
    public dialogRef: MatDialogRef<ProfilePlotModal>,
    @Inject(MAT_DIALOG_DATA) public data: ProfilePlotModalData
  ) {
    if (data) {
      this.annotation = data.annotation || new DataFrame();
    }
  }

  close() {
    this.dialogRef.close();
  }
}
