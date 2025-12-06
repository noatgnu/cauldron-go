import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {MatButtonModule} from "@angular/material/button";

@Component({
  selector: 'app-job-removal-modal',
  imports: [
    MatDialogModule,
    MatButtonModule
  ],
  templateUrl: './job-removal-modal.html',
  styleUrl: './job-removal-modal.scss',
})
export class JobRemovalModal {
  constructor(
    public dialogRef: MatDialogRef<JobRemovalModal>,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {}

  close() {
    this.dialogRef.close(false);
  }

  remove() {
    this.dialogRef.close(true);
  }
}
