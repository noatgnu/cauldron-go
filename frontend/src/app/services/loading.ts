import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

export interface LoadingState {
  isLoading: boolean;
  message: string;
}

@Injectable({
  providedIn: 'root'
})
export class LoadingService {
  private loadingSubject = new BehaviorSubject<LoadingState>({
    isLoading: false,
    message: ''
  });

  loading$ = this.loadingSubject.asObservable();

  show(message: string = 'Loading...') {
    this.loadingSubject.next({ isLoading: true, message });
  }

  hide() {
    this.loadingSubject.next({ isLoading: false, message: '' });
  }
}
