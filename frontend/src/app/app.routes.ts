import { Routes } from '@angular/router';
import { Home } from './home/home';
import { Settings } from './pages/settings/settings';
import { Jobs } from './pages/jobs/jobs';
import { JobDetail } from './pages/job-detail/job-detail';
import { Plugins } from './pages/plugins/plugins';
import { PluginList } from './pages/plugin-list/plugin-list';
import { PluginExecute } from './pages/plugin-execute/plugin-execute';
import { PluginEditor } from './pages/plugin-editor/plugin-editor';
import { PluginRegistry } from './pages/plugin-registry/plugin-registry';
import { PluginRegistryDetail } from './pages/plugin-registry-detail/plugin-registry-detail';
import { About } from './pages/about/about';
import { ParquetBrowser } from './pages/parquet-browser/parquet-browser';

export const routes: Routes = [
  { path: '', redirectTo: 'home', pathMatch: 'full' },
  { path: 'home', component: Home },
  { path: 'about', component: About },
  { path: 'settings', redirectTo: 'settings/general', pathMatch: 'full' },
  { path: 'settings/:section', component: Settings },
  { path: 'jobs', component: Jobs },
  { path: 'jobs/:id', component: JobDetail },
  { path: 'job/:id', component: JobDetail },
  { path: 'plugins', component: Plugins },
  { path: 'plugin-list', component: PluginList },
  { path: 'plugin-registry', component: PluginRegistry },
  { path: 'plugin-registry/:id', component: PluginRegistryDetail },
  { path: 'plugin/:id', component: PluginExecute },
  { path: 'plugin-editor', component: PluginEditor },
  { path: 'plugin-editor/new', component: PluginEditor },
  { path: 'plugin-editor/:id', component: PluginEditor },
  { path: 'parquet-browser', component: ParquetBrowser },
  { path: '**', redirectTo: '' }
];
