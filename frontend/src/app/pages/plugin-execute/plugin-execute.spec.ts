import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PluginExecute } from './plugin-execute';

describe('PluginExecute', () => {
  let component: PluginExecute;
  let fixture: ComponentFixture<PluginExecute>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PluginExecute]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginExecute);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
