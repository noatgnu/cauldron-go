import { ComponentFixture, TestBed } from '@angular/core/testing';
import { About } from './about';
import { Wails } from '../../core/services/wails';

describe('About', () => {
  let component: About;
  let fixture: ComponentFixture<About>;
  let wailsMock: any;

  beforeEach(async () => {
    wailsMock = {
      getLicenseInfo: vi.fn().mockResolvedValue({ go: [], npm: [] }),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };

    await TestBed.configureTestingModule({
      imports: [About],
      providers: [
        { provide: Wails, useValue: wailsMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(About);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should load license info on init', async () => {
    await fixture.whenStable();
    expect(wailsMock.getLicenseInfo).toHaveBeenCalled();
  });
});
