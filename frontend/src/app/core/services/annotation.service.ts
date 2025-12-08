import { Injectable } from '@angular/core';
import { Wails } from './wails';

export interface Annotation {
  sample: string;
  condition: string;
  batch?: string;
  color?: string;
}

// Kept for legacy compatibility in other components, but will be phased out.
export interface AnnotationColors {
  [condition: string]: string;
}

@Injectable({
  providedIn: 'root'
})
export class AnnotationService {
  private readonly DEFAULT_COLORS = [
    '#1976d2', '#388e3c', '#d32f2f', '#f57c00', '#7b1fa2',
    '#0097a7', '#c2185b', '#5d4037', '#fbc02d', '#00796b'
  ];

  private readonly ANNOTATION_FILENAMES = [
    'annotation.txt',
    'annotation.csv',
    'annotations.txt',
    'annotations.csv'
  ];

  constructor(private wails: Wails) {}

  async loadAnnotationsForJob(jobId: string): Promise<Annotation[]> {
    for (const filename of this.ANNOTATION_FILENAMES) {
      try {
        const annotationData = await this.wails.readJobOutputFile(jobId, filename);
        if (annotationData) {
          await this.wails.logToFile(`[AnnotationService] Loaded annotations from ${filename} for job ${jobId}`);
          return this.parseAnnotations(annotationData);
        }
      } catch {
        continue;
      }
    }
    await this.wails.logToFile(`[AnnotationService] No annotation file found for job ${jobId}`);
    return [];
  }

  parseAnnotations(data: string): Annotation[] {
    const lines = data.trim().split('\n').filter(line => !line.startsWith('#'));
    if (lines.length < 2) return []; // Must have header and at least one data row

    const headers = lines[0].toLowerCase().split(/[\t,]/);
    const sampleIdx = headers.findIndex(h => h.includes('sample'));
    const conditionIdx = headers.findIndex(h => h.includes('condition') || h.includes('group'));
    const batchIdx = headers.findIndex(h => h.includes('batch'));
    const colorIdx = headers.findIndex(h => h.includes('color'));

    if (sampleIdx === -1 || conditionIdx === -1) {
      return [];
    }

    const annotations: Annotation[] = [];
    for (let i = 1; i < lines.length; i++) {
      const values = lines[i].split(/[\t,]/);
      if (values.length > Math.max(sampleIdx, conditionIdx)) {
        const sample = values[sampleIdx]?.trim();
        const samplePath = sample.split(/[/\\]/).pop() || sample;
        const annotation: Annotation = {
          sample: samplePath,
          condition: values[conditionIdx]?.trim()
        };
        if (batchIdx !== -1 && values[batchIdx]) {
          annotation.batch = values[batchIdx].trim();
        }
        if (colorIdx !== -1 && values[colorIdx]) {
          annotation.color = values[colorIdx].trim();
        }
        annotations.push(annotation);
      }
    }
    return annotations;
  }

  async saveAnnotationsForJob(jobId: string, annotations: Annotation[], filename: string = 'annotation.txt'): Promise<void> {
    if (!annotations || annotations.length === 0) {
      await this.wails.logToFile(`[AnnotationService] No annotations to save for job ${jobId}`);
      return;
    }

    const hasBatch = annotations.some(a => a.batch);
    const hasColor = annotations.some(a => a.color);

    const headers = ['Sample', 'Condition'];
    if (hasBatch) headers.push('Batch');
    if (hasColor) headers.push('Color');

    const rows = annotations.map(a => {
      const row = [a.sample || '', a.condition || ''];
      if (hasBatch) {
        row.push(a.batch || '');
      }
      if (hasColor) {
        row.push(a.color || '');
      }
      return row.join('\t');
    });

    const content = [headers.join('\t'), ...rows].join('\n');

    await this.wails.writeJobOutputFile(jobId, filename, content);
    await this.wails.logToFile(`[AnnotationService] Saved ${annotations.length} annotations to ${filename} for job ${jobId}`);
  }

  getCondition(sampleName: string, annotations: Annotation[]): string {
    const annotation = annotations.find(a => a.sample === sampleName);
    return annotation ? annotation.condition : 'Unknown';
  }

  getDefaultColors(): string[] {
    return [...this.DEFAULT_COLORS];
  }
}
