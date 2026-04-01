import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal, WritableSignal } from '@angular/core';
import { LoadingScreenComponent } from './loading-screen';
import { LoadingService, LoadingState } from '../../services/loading';

describe('LoadingScreenComponent', () => {
  let component: LoadingScreenComponent;
  let fixture: ComponentFixture<LoadingScreenComponent>;
  let loadingSignal: WritableSignal<LoadingState>;

  beforeEach(async () => {
    loadingSignal = signal<LoadingState>({ isLoading: false, message: 'Loading...' });

    const mockLoadingService = {
      loading: loadingSignal
    };

    await TestBed.configureTestingModule({
      imports: [LoadingScreenComponent],
      providers: [
        { provide: LoadingService, useValue: mockLoadingService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(LoadingScreenComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should update loading state', async () => {
    loadingSignal.set({ isLoading: true, message: 'Test loading' });
    TestBed.flushEffects();
    await fixture.whenStable();
    expect(component.isLoading()).toBe(true);
    expect(component.loadingMessage()).toBe('Test loading');
  });
});
