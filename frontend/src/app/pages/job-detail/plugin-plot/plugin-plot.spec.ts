import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PluginPlot } from './plugin-plot';

describe('PluginPlot', () => {
  let component: PluginPlot;
  let fixture: ComponentFixture<PluginPlot>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PluginPlot]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginPlot);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
