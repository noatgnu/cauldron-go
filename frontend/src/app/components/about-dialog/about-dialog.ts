import { Component, OnInit, signal } from '@angular/core';
import { MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTabsModule } from '@angular/material/tabs';
import { CommonModule } from '@angular/common';
import { Wails } from '../../core/services/wails';

interface LicenseInfo {
  name: string;
  version: string;
  license: string;
  repository?: string;
}

interface LicenseData {
  go: LicenseInfo[];
  npm: LicenseInfo[];
}

@Component({
  selector: 'app-about-dialog',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatTabsModule
  ],
  templateUrl: './about-dialog.html',
  styleUrl: './about-dialog.scss',
})
export class AboutDialog implements OnInit {
  protected readonly version = '1.0.0';
  protected readonly copyright = 'Copyright 2025';
  protected readonly author = 'Toan K. Phung';
  protected readonly email = 'tphung001@dundee.ac.uk';
  protected readonly description = 'Proteomics data visualization tool';
  protected licenses = signal<LicenseData>({ go: [], npm: [] });
  protected loading = signal(true);

  constructor(private wails: Wails) {}

  async ngOnInit(): Promise<void> {
    try {
      const licenseData = await this.wails.getLicenseInfo();
      this.licenses.set(licenseData);
    } catch (error) {
      await this.wails.logToFile(`Failed to load license info: ${error}`);
    } finally {
      this.loading.set(false);
    }
  }
}
