import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PluginList } from './plugin-list';

describe('PluginList', () => {
  let component: PluginList;
  let fixture: ComponentFixture<PluginList>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PluginList]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginList);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
