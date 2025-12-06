import { ComponentFixture, TestBed } from '@angular/core/testing';

import { QfeaturesLimma } from './qfeatures-limma';

describe('QfeaturesLimma', () => {
  let component: QfeaturesLimma;
  let fixture: ComponentFixture<QfeaturesLimma>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [QfeaturesLimma]
    })
    .compileComponents();

    fixture = TestBed.createComponent(QfeaturesLimma);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
