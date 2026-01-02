import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CauldronLoaderComponent } from './cauldron-loader';

describe('CauldronLoaderComponent', () => {
  let component: CauldronLoaderComponent;
  let fixture: ComponentFixture<CauldronLoaderComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CauldronLoaderComponent]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CauldronLoaderComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should display default message', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.loader-message')?.textContent).toContain('Brewing...');
  });
});