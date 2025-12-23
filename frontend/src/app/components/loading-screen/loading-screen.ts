import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LoadingService } from '../../services/loading';

@Component({
  selector: 'app-loading-screen',
  templateUrl: './loading-screen.html',
  styleUrls: ['./loading-screen.scss'],
  imports: [CommonModule],
  standalone: true
})
export class LoadingScreenComponent {
  isLoading = false;
  loadingMessage = 'Loading...';

  constructor(private loadingService: LoadingService) {
    this.loadingService.loading$.subscribe(state => {
      this.isLoading = state.isLoading;
      this.loadingMessage = state.message;
    });
  }
}
