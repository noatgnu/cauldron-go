import { ComponentFixture, TestBed } from '@angular/core/testing';
import { BehaviorSubject } from 'rxjs';
import { LoadingScreenComponent } from './loading-screen';
import { LoadingService } from '../../services/loading';

describe('LoadingScreenComponent', () => {
  let component: LoadingScreenComponent;
  let fixture: ComponentFixture<LoadingScreenComponent>;
  let loadingSubject: BehaviorSubject<{ isLoading: boolean; message: string }>;

  beforeEach(async () => {
    loadingSubject = new BehaviorSubject<{ isLoading: boolean; message: string }>({ isLoading: false, message: 'Loading...' });

    const mockLoadingService = {
      loading$: loadingSubject.asObservable()
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

  it('should update loading state', () => {
    loadingSubject.next({ isLoading: true, message: 'Test loading' });
    expect(component.isLoading).toBe(true);
    expect(component.loadingMessage).toBe('Test loading');
  });
});
