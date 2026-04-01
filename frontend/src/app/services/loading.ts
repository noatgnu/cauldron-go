import { Injectable, signal, Signal } from '@angular/core';

export interface LoadingState {
  isLoading: boolean;
  message: string;
}

@Injectable({
  providedIn: 'root'
})
export class LoadingService {
  private _loading = signal<LoadingState>({
    isLoading: false,
    message: ''
  });

  loading: Signal<LoadingState> = this._loading.asReadonly();

  show(message: string = 'Loading...') {
    this._loading.set({ isLoading: true, message });
  }

  hide() {
    this._loading.set({ isLoading: false, message: '' });
  }
}
