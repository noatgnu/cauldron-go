import { Component, ChangeDetectionStrategy, effect, signal } from '@angular/core';
import { LoadingService } from '../../services/loading';

@Component({
  selector: 'app-loading-screen',
  templateUrl: './loading-screen.html',
  styleUrls: ['./loading-screen.scss'],
  imports: [],
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class LoadingScreenComponent {
  isLoading = signal(false);
  loadingMessage = signal('Loading...');

  constructor(private loadingService: LoadingService) {
    effect(() => {
      const state = this.loadingService.loading();
      this.isLoading.set(state.isLoading);
      this.loadingMessage.set(state.message);
    });
  }
}
