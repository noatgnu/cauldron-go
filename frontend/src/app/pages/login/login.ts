import { Component, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

@Component({
  selector: 'app-login',
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule
  ],
  templateUrl: './login.html',
  styleUrl: './login.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class Login {
  protected token = signal('');
  protected error = signal('');
  protected loading = signal(false);

  constructor(private router: Router) {}

  async submit(): Promise<void> {
    if (!this.token()) return;

    this.error.set('');
    this.loading.set(true);
    try {
      const res = await fetch('/auth/login', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: this.token() })
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        this.error.set(body.error || 'Invalid token');
        return;
      }
      await this.router.navigateByUrl('/');
    } catch {
      this.error.set('Failed to reach the server');
    } finally {
      this.loading.set(false);
    }
  }
}
