import { ComponentFixture, TestBed } from '@angular/core/testing';
import { EnvironmentIndicator } from './environment-indicator';

describe('EnvironmentIndicator', () => {
  let component: EnvironmentIndicator;
  let fixture: ComponentFixture<EnvironmentIndicator>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EnvironmentIndicator]
    })
    .compileComponents();

    fixture = TestBed.createComponent(EnvironmentIndicator);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
