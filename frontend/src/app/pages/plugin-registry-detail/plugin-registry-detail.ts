import { Component, OnInit, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatTabsModule } from '@angular/material/tabs';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTableModule } from '@angular/material/table';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatChipsModule } from '@angular/material/chips';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { ConfirmPluginInstallDialog, PluginInstallConfirmResult } from '../../components/confirm-plugin-install-dialog/confirm-plugin-install-dialog';
import { PluginInstallProgress } from '../../components/plugin-install-progress/plugin-install-progress';

interface RegistryPlugin {
  id: string;
  name: string;
  description: string;
  version: string;
  author: {
    id: number;
    name: string;
    email?: string;
  };
  category: {
    id: number;
    name: string;
    description?: string;
  };
  repository: string;
  commit_hash: string;
  icon?: string;
  requires_authentication: boolean;
  tags?: Array<{ id: number; name: string }>;
  created_at: string;
  updated_at: string;
  readme?: string;
  runtime?: {
    id: number;
    plugin: string;
    environments: string[];
    entrypoint: string;
  };
  inputs?: Array<{
    name: string;
    type: string;
    description: string;
    required: boolean;
    default?: string;
  }>;
  outputs?: Array<{
    name: string;
    type: string;
    description: string;
  }>;
  env_variables?: Array<{
    name: string;
    description: string;
    required: boolean;
    default?: string;
  }>;
}

@Component({
  selector: 'app-plugin-registry-detail',
  imports: [
    CommonModule,
    MatTabsModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatTableModule,
    MatDialogModule,
    MatChipsModule
  ],
  templateUrl: './plugin-registry-detail.html',
  styleUrl: './plugin-registry-detail.scss',
})
export class PluginRegistryDetail implements OnInit {
  protected plugin = signal<RegistryPlugin | null>(null);
  protected loading = signal(false);
  protected readmeHtml = signal<SafeHtml>('');
  protected isInstalled = signal(false);

  inputColumns = ['name', 'type', 'required', 'default', 'description'];
  outputColumns = ['name', 'type', 'description'];
  envColumns = ['name', 'required', 'default', 'description'];

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private wails: Wails,
    private dialog: MatDialog,
    private notification: NotificationService,
    private sanitizer: DomSanitizer,
    private pluginService: PluginV2Service
  ) {}

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.loadPluginDetail(id);
    } else {
      this.router.navigate(['/plugin-registry']);
    }
  }

  async loadPluginDetail(id: string): Promise<void> {
    this.loading.set(true);
    try {
      const plugin = await this.wails.getRegistryPlugin(id) as RegistryPlugin;
      this.plugin.set(plugin);

      await this.checkIfInstalled(plugin);

      if (plugin.readme) {
        try {
          const html = this.convertMarkdownToHtml(plugin.readme);
          this.readmeHtml.set(this.sanitizer.bypassSecurityTrustHtml(html));
        } catch (mdError) {
          await this.wails.logToFile(`[PluginRegistryDetail] Failed to parse README: ${mdError}`);
          this.readmeHtml.set('');
        }
      }
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistryDetail] Failed to load plugin: ${error}`);
      this.notification.showError('Failed to load plugin details');
      this.router.navigate(['/plugin-registry']);
    } finally {
      this.loading.set(false);
    }
  }

  private async checkIfInstalled(plugin: RegistryPlugin): Promise<void> {
    if (!plugin.repository) {
      this.isInstalled.set(false);
      return;
    }

    try {
      const installedPlugins = await this.pluginService.getAllPlugins();
      const normalizedRegistryRepo = this.normalizeRepoUrl(plugin.repository);

      const installed = installedPlugins.some(p => {
        if (!p.repository) return false;
        return this.normalizeRepoUrl(p.repository) === normalizedRegistryRepo;
      });

      this.isInstalled.set(installed);
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistryDetail] Failed to check installation status: ${error}`);
      this.isInstalled.set(false);
    }
  }

  private normalizeRepoUrl(url: string): string {
    return url.toLowerCase().replace(/\.git$/, '').replace(/\/$/, '');
  }

  convertMarkdownToHtml(markdown: string): string {
    if (!markdown || typeof markdown !== 'string') {
      return '';
    }

    try {
      let html = markdown;

      html = html.replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');

      html = this.parseMarkdownTables(html);

      html = html.replace(/^#{4}\s+(.*$)/gim, '<h4>$1</h4>')
        .replace(/^#{3}\s+(.*$)/gim, '<h3>$1</h3>')
        .replace(/^#{2}\s+(.*$)/gim, '<h2>$1</h2>')
        .replace(/^#{1}\s+(.*$)/gim, '<h1>$1</h1>');

      html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');

      html = html.replace(/^\*\s+(.*)$/gim, '<li>$1</li>')
        .replace(/^-\s+(.*)$/gim, '<li>$1</li>')
        .replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>');

      html = html.replace(/^\d+\.\s+(.*)$/gim, '<li>$1</li>')
        .replace(/(<li>.*<\/li>\n?)+/g, (match) => {
          if (!match.includes('<ul>')) {
            return '<ol>' + match + '</ol>';
          }
          return match;
        });

      html = html.replace(/```([^`]*?)```/gs, '<pre><code>$1</code></pre>');

      html = html.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>')
        .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
        .replace(/\*(.+?)\*/g, '<em>$1</em>')
        .replace(/___(.+?)___/g, '<strong><em>$1</em></strong>')
        .replace(/__(.+?)__/g, '<strong>$1</strong>')
        .replace(/_(.+?)_/g, '<em>$1</em>');

      html = html.replace(/`([^`]+)`/g, '<code>$1</code>');

      html = html.replace(/\n\n+/g, '</p><p>')
        .replace(/\n/g, '<br>');

      html = html.replace(/<br><ul>/g, '<ul>')
        .replace(/<\/ul><br>/g, '</ul>')
        .replace(/<br><ol>/g, '<ol>')
        .replace(/<\/ol><br>/g, '</ol>')
        .replace(/<br><table>/g, '<table>')
        .replace(/<\/table><br>/g, '</table>')
        .replace(/<br><pre>/g, '<pre>')
        .replace(/<\/pre><br>/g, '</pre>');

      return `<div>${html}</div>`;
    } catch (error) {
      return `<p>Error rendering README</p>`;
    }
  }

  private parseMarkdownTables(markdown: string): string {
    const lines = markdown.split('\n');
    const result: string[] = [];
    let inTable = false;
    let tableRows: string[] = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim();

      if (line.includes('|') && line.split('|').length >= 3) {
        if (!inTable) {
          inTable = true;
          tableRows = [];
        }
        tableRows.push(line);
      } else {
        if (inTable) {
          result.push(this.convertTableToHtml(tableRows));
          inTable = false;
          tableRows = [];
        }
        result.push(lines[i]);
      }
    }

    if (inTable) {
      result.push(this.convertTableToHtml(tableRows));
    }

    return result.join('\n');
  }

  private convertTableToHtml(tableRows: string[]): string {
    if (tableRows.length < 2) {
      return tableRows.join('\n');
    }

    const headerRow = tableRows[0].split('|').filter(cell => cell.trim() !== '');
    const separatorRow = tableRows[1];

    if (!separatorRow.match(/^[\s|:-]+$/)) {
      return tableRows.join('\n');
    }

    const dataRows = tableRows.slice(2);

    let html = '<table class="markdown-table">';

    html += '<thead><tr>';
    headerRow.forEach(cell => {
      html += `<th>${cell.trim()}</th>`;
    });
    html += '</tr></thead>';

    html += '<tbody>';
    dataRows.forEach(row => {
      const cells = row.split('|').filter(cell => cell.trim() !== '');
      html += '<tr>';
      cells.forEach(cell => {
        html += `<td>${cell.trim()}</td>`;
      });
      html += '</tr>';
    });
    html += '</tbody>';

    html += '</table>';

    return html;
  }

  formatDate(dateString: string): string {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    });
  }

  goBack(): void {
    this.router.navigate(['/plugin-registry']);
  }

  async installPlugin(): Promise<void> {
    try {
      await this.wails.logToFile('========================================');
      await this.wails.logToFile('[PluginRegistryDetail] Install button clicked');

      const plugin = this.plugin();
      if (!plugin || !plugin.repository) {
        this.notification.showError('This plugin does not have a repository URL');
        return;
      }

      await this.wails.logToFile(`[PluginRegistryDetail] Plugin name: ${plugin.name}, repository: ${plugin.repository}`);
      await this.wails.logToFile(`[PluginRegistryDetail] Fetching plugin dependencies for: ${plugin.name}`);

      let hasPythonDeps = false;
      let hasRDeps = false;
      let runtimeEnvironments: string[] = [];

      try {
        const deps = await this.wails.fetchPluginDependencies(plugin.repository);
        await this.wails.logToFile(`[PluginRegistryDetail] Raw deps response: ${JSON.stringify(deps)}`);
        hasPythonDeps = deps['hasPythonDeps'] === true;
        hasRDeps = deps['hasRDeps'] === true;
        runtimeEnvironments = deps['runtimeEnvironments'] || [];
        await this.wails.logToFile(`[PluginRegistryDetail] After assignment - Python: ${hasPythonDeps}, R: ${hasRDeps}, envs: ${runtimeEnvironments.join(',')}`);
      } catch (error) {
        await this.wails.logToFile(`[PluginRegistryDetail] Failed to fetch dependencies: ${error}`);
        if (plugin.runtime?.environments) {
          runtimeEnvironments = plugin.runtime.environments;
        }
      }

      await this.wails.logToFile(`[PluginRegistryDetail] Opening install dialog for: ${plugin.name}`);
      await this.wails.logToFile(`[PluginRegistryDetail] Dialog data - hasPythonDeps: ${hasPythonDeps}, hasRDeps: ${hasRDeps}, runtimeEnvs: ${runtimeEnvironments.join(', ')}`);

      const dialogRef = this.dialog.open(ConfirmPluginInstallDialog, {
        width: '600px',
        disableClose: true,
        data: {
          repo: plugin.repository,
          ref: plugin.commit_hash,
          name: plugin.name,
          id: plugin.id,
          version: plugin.version,
          author: plugin.author?.name || 'Unknown',
          description: plugin.description,
          category: plugin.category?.name || 'Uncategorized',
          requiresAuthentication: plugin.requires_authentication,
          runtimeEnvironments,
          hasPythonDeps,
          hasRDeps
        }
      });

      const result: PluginInstallConfirmResult | undefined = await dialogRef.afterClosed().toPromise();

      if (result?.confirmed) {
        const progressRef = this.dialog.open(PluginInstallProgress, {
          width: '500px',
          disableClose: true,
          data: {
            repoURL: plugin.repository,
            commitHash: plugin.commit_hash,
            sshKeyPath: result.sshKeyPath,
            passphrase: result.passphrase,
            createVenv: result.createVenv,
            basePythonPath: result.basePythonPath,
            createRenv: result.createRenv,
            renvName: result.renvName
          }
        });

        try {
          await progressRef.afterClosed().toPromise();
          await this.checkIfInstalled(plugin);
          this.notification.showSuccess('Plugin installed successfully');
        } catch (error) {
          await this.wails.logToFile(`[PluginRegistryDetail] Installation failed: ${error}`);
          this.notification.showError('Failed to install plugin');
        }
      }
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistryDetail] Error in installPlugin: ${error}`);
      this.notification.showError('Failed to open install dialog');
    }
  }

  trackByTagId(index: number, tag: { id: number; name: string }): number {
    return tag.id;
  }
}
