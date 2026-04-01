import { Component, Inject, OnInit, ChangeDetectionStrategy } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatTooltipModule } from '@angular/material/tooltip';
import { CommonModule } from '@angular/common';
import { NotificationService } from '../../core/services/notification.service';

export interface TableColumn {
  name: string;
  label: string;
  type?: 'text' | 'number';
  required?: boolean;
  description?: string;
}

export interface GenericTableEditorData {
  columns: TableColumn[];
  data?: any[];
  title?: string;
  mode: 'edit' | 'create';
}

@Component({
  selector: 'app-generic-table-editor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatDialogModule,
    MatButtonModule,
    MatInputModule,
    MatFormFieldModule,
    MatIconModule,
    MatTableModule,
    MatPaginatorModule,
    MatTooltipModule
  ],
  templateUrl: './generic-table-editor.html',
  styleUrls: ['./generic-table-editor.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class GenericTableEditor implements OnInit {
  tableData: any[] = [];
  columns: TableColumn[] = [];
  displayedColumns: string[] = [];
  title: string = 'Table Editor';
  mode: 'edit' | 'create' = 'edit';

  pagedData: any[] = [];
  pageSize: number = 10;
  pageIndex: number = 0;
  totalRows: number = 0;

  constructor(
    public dialogRef: MatDialogRef<GenericTableEditor>,
    @Inject(MAT_DIALOG_DATA) public data: GenericTableEditorData,
    private notificationService: NotificationService
  ) {
    this.columns = data.columns || [];
    this.displayedColumns = [...this.columns.map(c => c.name), 'actions'];
    this.title = data.title || 'Table Editor';
    this.mode = data.mode || 'edit';

    if (data.data && data.data.length > 0) {
      this.tableData = data.data.map(row => ({...row}));
    } else {
      this.addRow();
    }

    this.updatePagedData();
  }

  ngOnInit(): void {
    this.updatePagedData();
  }

  updatePagedData(): void {
    this.totalRows = this.tableData.length;
    const startIndex = this.pageIndex * this.pageSize;
    const endIndex = startIndex + this.pageSize;
    this.pagedData = this.tableData.slice(startIndex, endIndex);
  }

  onPageChange(event: PageEvent): void {
    this.pageIndex = event.pageIndex;
    this.pageSize = event.pageSize;
    this.updatePagedData();
  }

  addRow(): void {
    const newRow: any = {};
    this.columns.forEach(col => {
      newRow[col.name] = '';
    });
    this.tableData.push(newRow);
    this.updatePagedData();

    const lastPage = Math.floor((this.totalRows - 1) / this.pageSize);
    if (this.pageIndex !== lastPage) {
      this.pageIndex = lastPage;
      this.updatePagedData();
    }
  }

  deleteRow(index: number): void {
    const actualIndex = this.pageIndex * this.pageSize + index;
    this.tableData.splice(actualIndex, 1);

    if (this.pagedData.length === 1 && this.pageIndex > 0) {
      this.pageIndex--;
    }
    this.updatePagedData();
  }

  duplicateRow(index: number): void {
    const actualIndex = this.pageIndex * this.pageSize + index;
    const rowToCopy = {...this.tableData[actualIndex]};
    this.tableData.splice(actualIndex + 1, 0, rowToCopy);
    this.updatePagedData();
  }

  isValid(): boolean {
    for (const row of this.tableData) {
      for (const col of this.columns) {
        if (col.required && (!row[col.name] || row[col.name].toString().trim() === '')) {
          return false;
        }
      }
    }
    return this.tableData.length > 0;
  }

  save(): void {
    if (!this.isValid()) {
      this.notificationService.showError('Please fill in all required fields');
      return;
    }
    this.dialogRef.close(this.tableData);
  }

  cancel(): void {
    this.dialogRef.close(null);
  }

  importFromFile(): void {
    this.notificationService.showInfo('Import functionality will be added');
  }

  exportToFile(): void {
    this.notificationService.showInfo('Export functionality will be added');
  }
}
