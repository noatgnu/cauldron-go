import { Injectable, signal } from '@angular/core';
import { Wails } from './wails';

export interface Annotation {
  sample: string;
  condition: string;
  batch?: string;
}

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
    const lines = data.trim().split('\n');
    if (lines.length === 0) return [];

    const headers = lines[0].split(/[\t,]/);
    const sampleIdx = headers.findIndex(h => h.toLowerCase().includes('sample'));
    const conditionIdx = headers.findIndex(h => h.toLowerCase().includes('condition') || h.toLowerCase().includes('group'));
    const batchIdx = headers.findIndex(h => h.toLowerCase().includes('batch'));

    if (sampleIdx === -1 || conditionIdx === -1) {
      return [];
    }

    const annotations: Annotation[] = [];
    for (let i = 1; i < lines.length; i++) {
      const values = lines[i].split(/[\t,]/);
      if (values.length > Math.max(sampleIdx, conditionIdx)) {
        const sample = values[sampleIdx].trim();
        const samplePath = sample.split(/[/\\]/).pop() || sample;
        const annotation: Annotation = {
          sample: samplePath,
          condition: values[conditionIdx].trim()
        };
        if (batchIdx !== -1 && values[batchIdx]) {
          annotation.batch = values[batchIdx].trim();
        }
        annotations.push(annotation);
      }
    }

    return annotations;
  }

  async saveAnnotationsForJob(jobId: string, annotations: Annotation[], filename: string = 'annotation.txt'): Promise<void> {
    const headers = annotations[0]?.batch !== undefined
      ? ['Sample', 'Condition', 'Batch']
      : ['Sample', 'Condition'];

    const rows = annotations.map(a => {
      const row = [a.sample, a.condition];
      if (a.batch !== undefined) {
        row.push(a.batch);
      }
      return row.join('\t');
    });

    const content = [headers.join('\t'), ...rows].join('\n');

    await this.wails.writeJobOutputFile(jobId, filename, content);
    await this.wails.logToFile(`[AnnotationService] Saved annotations to ${filename} for job ${jobId}`);
  }

  async loadConditionColorsForJob(jobId: string): Promise<AnnotationColors> {
    try {
      const colorData = await this.wails.readJobOutputFile(jobId, 'annotation_colors.json');
      if (colorData) {
        return JSON.parse(colorData);
      }
    } catch (err) {
      await this.wails.logToFile(`[AnnotationService] No color mapping found for job ${jobId}`);
    }
    return {};
  }

  async saveConditionColorsForJob(jobId: string, colors: AnnotationColors): Promise<void> {
    await this.wails.writeJobOutputFile(jobId, 'annotation_colors.json', JSON.stringify(colors, null, 2));
    await this.wails.logToFile(`[AnnotationService] Saved condition colors for job ${jobId}`);
  }

  getCondition(sampleName: string, annotations: Annotation[]): string {
    const annotation = annotations.find(a => a.sample === sampleName);
    return annotation ? annotation.condition : 'Unknown';
  }

  getDefaultColors(): string[] {
    return [...this.DEFAULT_COLORS];
  }

  assignDefaultColors(conditions: string[], existingColors: AnnotationColors = {}): AnnotationColors {
    const colors: AnnotationColors = { ...existingColors };
    let colorIndex = 0;

    for (const condition of conditions) {
      if (!colors[condition]) {
        colors[condition] = this.DEFAULT_COLORS[colorIndex % this.DEFAULT_COLORS.length];
        colorIndex++;
      }
    }

    return colors;
  }

  extractSampleNames(samplePaths: string[]): string[] {
    return samplePaths.map(path => path.split(/[/\\]/).pop() || path);
  }
}
