import { Component, Inject, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';

export interface BoundPlugin {

  name: string;

  description: string;

  id: string;

}



export interface BoundPluginsDialogData {

  envName: string;

  plugins: BoundPlugin[];

}



@Component({

  selector: 'app-bound-plugins-dialog',

  standalone: true,

  imports: [

    CommonModule,

    MatDialogModule,

    MatButtonModule,

    MatListModule,

    MatIconModule

  ],

  templateUrl: './bound-plugins-dialog.html',

  styleUrl: './bound-plugins-dialog.scss',

  changeDetection: ChangeDetectionStrategy.OnPush

})

export class BoundPluginsDialogComponent {

  constructor(

    public dialogRef: MatDialogRef<BoundPluginsDialogComponent>,

    @Inject(MAT_DIALOG_DATA) public data: BoundPluginsDialogData

  ) {}



  close(): void {

    this.dialogRef.close();

  }

}
