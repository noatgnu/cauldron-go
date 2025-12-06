import {Component, Inject, Input} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule} from "@angular/forms";
import {Wails} from "../../core/services/wails";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatSelectModule} from "@angular/material/select";
import {MatProgressBarModule} from "@angular/material/progress-bar";
import {VolcanoPlot} from "../volcano-plot/volcano-plot";
import {DataFrame, fromCSV, IDataFrame} from "data-forge";
import {VolcanoDataRow} from "../volcano-plot/volcano-data-row";

export interface VolcanoPlotModalData {
  differentialAnalysisFile: string;
  log2FoldChangeColumn: string;
  pValueColumn: string;
  comparisonColumn: string;
  indexCols: string[];
}

@Component({
  selector: 'app-volcano-plot-modal',
  imports: [
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatSelectModule,
    MatProgressBarModule,
    ReactiveFormsModule,
    VolcanoPlot
  ],
  templateUrl: './volcano-plot-modal.html',
  styleUrl: './volcano-plot-modal.scss',
})
export class VolcanoPlotModal {
  _differentialAnalysisFile: string = '';
  log2FoldChangeColumn: string = 'logFC';
  pValueColumn: string = 'adj.P.Val';
  comparisonColumn: string = 'comparison';
  _indexCols: string[] = [];

  comparisons: string[] = [];
  dfFile: IDataFrame<number, any> = new DataFrame();
  df: IDataFrame<number, VolcanoDataRow> = new DataFrame();

  form!: FormGroup;
  assembling: boolean = false;
  revision: number = 0;

  constructor(
    public dialogRef: MatDialogRef<VolcanoPlotModal>,
    private wails: Wails,
    private fb: FormBuilder,
    @Inject(MAT_DIALOG_DATA) public data: VolcanoPlotModalData
  ) {
    if (data) {
      this._differentialAnalysisFile = data.differentialAnalysisFile || '';
      this.log2FoldChangeColumn = data.log2FoldChangeColumn || 'logFC';
      this.pValueColumn = data.pValueColumn || 'adj.P.Val';
      this.comparisonColumn = data.comparisonColumn || 'comparison';
      this._indexCols = data.indexCols || [];
    }

    this.form = this.fb.group({
      pValueCutoff: new FormControl(0.05),
      log2FoldChangeCutoff: new FormControl(0.6),
      pValueColumn: new FormControl(this.pValueColumn),
      log2FoldChangeColumn: new FormControl(this.log2FoldChangeColumn),
      comparison: new FormControl(''),
      indexCols: new FormControl(this._indexCols)
    });

    this.form.controls['comparison'].valueChanges.subscribe(() => {
      this.updatePlotData();
    });

    if (this._differentialAnalysisFile) {
      this.loadFile(this._differentialAnalysisFile);
    }
  }

  async loadFile(filePath: string) {
    try {
      this.assembling = true;
      const fileContent = await this.wails.readFile(filePath);

      const df = fromCSV(fileContent);
      this.dfFile = df;

      this.comparisons = this.dfFile.getSeries(this.comparisonColumn).distinct().toArray();

      if (this.comparisons.length > 0 && !this.form.controls['comparison'].value) {
        this.form.controls['comparison'].setValue(this.comparisons[0]);
      }

      this.updatePlotData();
      this.assembling = false;
    } catch (error) {
      console.error('Failed to load differential analysis file:', error);
      this.assembling = false;
    }
  }

  updatePlotData() {
    const comparison = this.form.controls['comparison'].value;
    if (!comparison) return;

    const data: VolcanoDataRow[] = [];

    this.dfFile.where((row: any) => {
      return row[this.comparisonColumn] === comparison;
    }).forEach((row: any) => {
      if (this.form.controls['indexCols'].value) {
        const indexCols = this.form.controls['indexCols'].value;
        const newRow: VolcanoDataRow = {
          label: row[indexCols[0]],
          index: indexCols.map((col: any) => row[col]).join("|"),
          x: parseFloat(row[this.log2FoldChangeColumn]),
          y: -Math.log10(parseFloat(row[this.pValueColumn]))
        };
        data.push(newRow);
      }
    });

    this.df = new DataFrame(data);
    this.revision++;
  }

  get pValueCutoffTransformed(): number {
    return -Math.log10(this.form.value.pValueCutoff || 0.05);
  }

  close() {
    this.dialogRef.close();
  }
}
