import { Component, Input, OnInit, signal, computed, inject, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import { DynamicFormComponent } from '../dynamic-form/dynamic-form';
import { BatchJobRowDialog, BatchJobRowDialogData } from '../batch-job-row-dialog/batch-job-row-dialog';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';

interface BatchRow {
  id: string;
  values: Record<string, any>;
}

@Component({
  selector: 'app-batch-job-builder',
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatTableModule,
    MatFormFieldModule,
    MatSelectModule,
    MatTooltipModule,
    DynamicFormComponent
  ],
  templateUrl: './batch-job-builder.html',
  styleUrl: './batch-job-builder.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class BatchJobBuilder implements OnInit {
  @Input() plugin!: models.PluginV2;

  private dialog = inject(MatDialog);
  private pluginService = inject(PluginV2Service);
  private wails = inject(Wails);
  private notification = inject(NotificationService);
  private router = inject(Router);

  rows = signal<BatchRow[]>([]);
  columns = ['summary', 'actions'];
  creating = signal(false);
  bulkFieldName = signal<string | null>(null);
  bulkValue = signal<Record<string, any>>({});
  targetFileInputName = signal<string | null>(null);

  fileInputs = computed(() => (this.plugin?.definition.inputs || []).filter(i => i.type === 'file'));

  bulkApplyPlugin = computed<models.PluginV2 | null>(() => {
    const fieldName = this.bulkFieldName();
    if (!fieldName || !this.plugin) return null;
    const input = this.plugin.definition.inputs.find(i => i.name === fieldName);
    if (!input) return null;
    return new models.PluginV2({
      ...this.plugin,
      definition: new models.PluginDefinition({
        ...this.plugin.definition,
        inputs: [input]
      })
    });
  });

  ngOnInit() {
    const firstFileInput = this.fileInputs()[0];
    if (firstFileInput) {
      this.targetFileInputName.set(firstFileInput.name);
    }
    this.addEmptyRow();
  }

  rowSummary(row: BatchRow): string {
    const fileInputName = this.targetFileInputName();
    if (fileInputName && row.values[fileInputName]) {
      const value = String(row.values[fileInputName]);
      const parts = value.split(/[\\/]/);
      return parts[parts.length - 1] || value;
    }
    const index = this.rows().findIndex(r => r.id === row.id);
    return `Job ${index + 1}`;
  }

  addEmptyRow() {
    const first = this.rows()[0];
    this.rows.update(rows => [...rows, { id: crypto.randomUUID(), values: first ? { ...first.values } : {} }]);
  }

  duplicateRow(id: string) {
    const row = this.rows().find(r => r.id === id);
    if (!row) return;
    this.rows.update(rows => [...rows, { id: crypto.randomUUID(), values: { ...row.values } }]);
  }

  removeRow(id: string) {
    this.rows.update(rows => rows.filter(r => r.id !== id));
  }

  async editRow(id: string) {
    const row = this.rows().find(r => r.id === id);
    if (!row) return;

    const dialogRef = this.dialog.open<BatchJobRowDialog, BatchJobRowDialogData, Record<string, any>>(BatchJobRowDialog, {
      width: '900px',
      maxHeight: '90vh',
      data: { plugin: this.plugin, initialValues: row.values }
    });

    const result = await firstValueFrom(dialogRef.afterClosed());
    if (result) {
      this.rows.update(rows => rows.map(r => (r.id === id ? { ...r, values: result } : r)));
    }
  }

  async addRowsFromFiles() {
    const fileInputName = this.targetFileInputName();
    if (!fileInputName) {
      this.notification.showError('This plugin has no file input to add files for.');
      return;
    }

    let paths: string[];
    try {
      paths = await this.wails.openMultipleFilesDialog('Select Files');
    } catch (err) {
      await this.wails.logToFile(`[BatchJobBuilder] Failed to open multi-file dialog: ${err}`);
      return;
    }
    if (!paths || paths.length === 0) return;

    const first = this.rows()[0];
    const baseValues = first ? { ...first.values } : {};
    const newRows: BatchRow[] = paths.map(path => ({
      id: crypto.randomUUID(),
      values: { ...baseValues, [fileInputName]: path }
    }));

    this.rows.update(rows => [...rows, ...newRows]);
  }

  onBulkFieldChange(fieldName: string) {
    this.bulkFieldName.set(fieldName || null);
    this.bulkValue.set({});
  }

  onBulkFormChange(values: Record<string, any>) {
    this.bulkValue.set(values);
  }

  applyBulkValue() {
    const fieldName = this.bulkFieldName();
    if (!fieldName) return;
    const value = this.bulkValue()[fieldName];
    this.rows.update(rows => rows.map(r => ({ ...r, values: { ...r.values, [fieldName]: value } })));
    this.notification.showSuccess(`Applied to ${this.rows().length} job(s).`);
  }

  async createBatch() {
    if (this.rows().length === 0) {
      this.notification.showError('Add at least one job to the batch.');
      return;
    }

    this.creating.set(true);
    try {
      const jobs = this.rows().map(r => r.values);
      const batchId = await this.pluginService.executeBatch(this.plugin.id, this.plugin.definition.plugin.name, jobs);
      this.notification.showSuccess(`Batch created with ${jobs.length} job(s).`);
      await this.router.navigate(['/job-batch', batchId]);
    } catch (err) {
      this.notification.showError(`Failed to create batch: ${err}`);
      await this.wails.logToFile(`[BatchJobBuilder] Failed to create batch: ${err}`);
    } finally {
      this.creating.set(false);
    }
  }
}
