import { Component, Inject, ViewChild, ChangeDetectionStrategy } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { DynamicFormComponent } from '../dynamic-form/dynamic-form';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';

export interface BatchJobRowDialogData {
  plugin: models.PluginV2;
  initialValues: Record<string, any>;
}

@Component({
  selector: 'app-batch-job-row-dialog',
  imports: [MatDialogModule, MatButtonModule, DynamicFormComponent],
  templateUrl: './batch-job-row-dialog.html',
  styleUrl: './batch-job-row-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class BatchJobRowDialog {
  @ViewChild(DynamicFormComponent) dynamicForm?: DynamicFormComponent;

  constructor(
    private dialogRef: MatDialogRef<BatchJobRowDialog, Record<string, any>>,
    @Inject(MAT_DIALOG_DATA) public data: BatchJobRowDialogData
  ) {}

  ngAfterViewInit() {
    setTimeout(async () => {
      await this.dynamicForm?.loadFromJobParameters(this.data.initialValues);
    }, 500);
  }

  save() {
    this.dynamicForm?.submit();
  }

  onFormSubmit(values: Record<string, any>) {
    this.dialogRef.close(values);
  }

  cancel() {
    this.dialogRef.close();
  }
}
