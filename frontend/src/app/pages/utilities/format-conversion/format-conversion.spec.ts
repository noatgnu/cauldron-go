import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FormatConversion } from './format-conversion';

describe('FormatConversion', () => {
  let component: FormatConversion;
  let fixture: ComponentFixture<FormatConversion>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FormatConversion]
    })
    .compileComponents();

    fixture = TestBed.createComponent(FormatConversion);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
