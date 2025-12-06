import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Alphastats } from './alphastats';

describe('Alphastats', () => {
  let component: Alphastats;
  let fixture: ComponentFixture<Alphastats>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Alphastats]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Alphastats);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
