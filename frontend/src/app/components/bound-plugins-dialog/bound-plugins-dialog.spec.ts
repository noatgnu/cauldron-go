import { ComponentFixture, TestBed } from '@angular/core/testing';

import { BoundPluginsDialog } from './bound-plugins-dialog';

describe('BoundPluginsDialog', () => {
  let component: BoundPluginsDialog;
  let fixture: ComponentFixture<BoundPluginsDialog>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [BoundPluginsDialog]
    })
    .compileComponents();

    fixture = TestBed.createComponent(BoundPluginsDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
