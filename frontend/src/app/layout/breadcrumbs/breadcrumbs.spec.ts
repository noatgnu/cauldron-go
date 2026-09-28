import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Breadcrumbs } from './breadcrumbs';
import { Router, ActivatedRoute } from '@angular/router';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { Wails } from '../../core/services/wails';
import { vi } from 'vitest';
import { of } from 'rxjs';

describe('Breadcrumbs', () => {
  let component: Breadcrumbs;
  let fixture: ComponentFixture<Breadcrumbs>;
  let routerMock: any;
  let activatedRouteMock: any;
  let pluginV2ServiceMock: any;
  let wailsMock: any;

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
    wailsMock = {
      getRegistryRecipe: vi.fn().mockResolvedValue({ label: 'Registry Recipe' })
    };

    await TestBed.configureTestingModule({
      imports: [Breadcrumbs],
      providers: [
        { provide: Router, useValue: routerMock },
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: PluginV2Service, useValue: pluginV2ServiceMock },
        { provide: Wails, useValue: wailsMock }
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

  it('shows only Home for the home page, not a duplicate crumb', async () => {
    routerMock.url = '/home';
    await component['updateBreadcrumbs']();
    expect(component['breadcrumbs']()).toEqual([{ label: 'Home', url: '/' }]);
  });

  it('shows only Home for the root path', async () => {
    routerMock.url = '/';
    await component['updateBreadcrumbs']();
    expect(component['breadcrumbs']()).toEqual([{ label: 'Home', url: '/' }]);
  });

  it('still builds crumbs for other routes', async () => {
    routerMock.url = '/jobs';
    await component['updateBreadcrumbs']();
    expect(component['breadcrumbs']()).toEqual([
      { label: 'Home', url: '/' },
      { label: 'Jobs', url: '/jobs' }
    ]);
  });

  it('labels the recipe registry list route', async () => {
    routerMock.url = '/recipe-registry';
    await component['updateBreadcrumbs']();
    expect(component['breadcrumbs']()).toEqual([
      { label: 'Home', url: '/' },
      { label: 'Recipe Registry', url: '/recipe-registry' }
    ]);
  });

  it('resolves the registry recipe label for a recipe-registry detail route', async () => {
    routerMock.url = '/recipe-registry/11111111-1111-1111-1111-111111111111';
    await component['updateBreadcrumbs']();
    expect(wailsMock.getRegistryRecipe).toHaveBeenCalledWith('11111111-1111-1111-1111-111111111111');
    expect(component['breadcrumbs']()).toEqual([
      { label: 'Home', url: '/' },
      { label: 'Recipe Registry', url: '/recipe-registry' },
      { label: 'Registry Recipe', url: '/recipe-registry/11111111-1111-1111-1111-111111111111' }
    ]);
  });
});
