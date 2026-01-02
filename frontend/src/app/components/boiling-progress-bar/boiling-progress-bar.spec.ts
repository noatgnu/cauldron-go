import { ComponentFixture, TestBed } from '@angular/core/testing';
import { BoilingProgressBarComponent } from './boiling-progress-bar';

describe('BoilingProgressBarComponent', () => {
  let component: BoilingProgressBarComponent;
  let fixture: ComponentFixture<BoilingProgressBarComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [BoilingProgressBarComponent]
    })
    .compileComponents();

    fixture = TestBed.createComponent(BoilingProgressBarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should have bubbles initialized', () => {
    expect(component.bubbles.length).toBeGreaterThan(0);
  });
});