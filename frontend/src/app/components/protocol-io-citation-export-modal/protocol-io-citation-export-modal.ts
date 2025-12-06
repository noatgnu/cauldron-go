import {Component} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule} from "@angular/forms";
import {MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";

export interface ProtocolIoCitationExportConfig {
  idList: string;
}

@Component({
  selector: 'app-protocol-io-citation-export-modal',
  imports: [
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule
  ],
  templateUrl: './protocol-io-citation-export-modal.html',
  styleUrl: './protocol-io-citation-export-modal.scss',
})
export class ProtocolIoCitationExportModal {
  form!: FormGroup;

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<ProtocolIoCitationExportModal>
  ) {
    this.form = this.fb.group({
      idList: new FormControl<string>("")
    });
  }

  close() {
    this.dialogRef.close();
  }

  download() {
    this.dialogRef.close(this.form.value);
  }
}
