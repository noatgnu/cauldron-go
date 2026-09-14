import { Component, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { Router } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatMenuModule } from '@angular/material/menu';
import { Wails } from '../../core/services/wails';

@Component({
  selector: 'app-quick-nav',
  imports: [MatIconModule, MatButtonModule, MatMenuModule],
  templateUrl: './quick-nav.html',
  styleUrl: './quick-nav.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class QuickNav implements OnInit {
  protected visible = signal(false);

  constructor(private router: Router, private wails: Wails) {}

  async ngOnInit(): Promise<void> {
    try {
      const capabilities = await this.wails.getRuntimeCapabilities();
      this.visible.set(capabilities.serverMode === true);
    } catch {
      this.visible.set(false);
    }
  }

  navigate(route: string): void {
    this.router.navigate([route]);
  }
}
