import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Breadcrumbs } from './breadcrumbs';
import { Router, ActivatedRoute } from '@angular/router';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { vi } from 'vitest';
import { of } from 'rxjs';

describe('Breadcrumbs', () => {
  let component: Breadcrumbs;
  let fixture: ComponentFixture<Breadcrumbs>;
  let routerMock: any;
  let activatedRouteMock: any;
  let pluginV2ServiceMock: any;

  beforeEach(async () => {
    routerMock = {
      events: of([]),
      url: '/',
      navigate: vi.fn()
    };
    activatedRouteMock = {};
    pluginV2ServiceMock = {
      getPlugin: vi.fn().mockResolvedValue({ definition: { plugin: { name: 'Test Plugin' } } })
    };

    await TestBed.configureTestingModule({
      imports: [Breadcrumbs],
      providers: [
        { provide: Router, useValue: routerMock },
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: PluginV2Service, useValue: pluginV2ServiceMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Breadcrumbs);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
