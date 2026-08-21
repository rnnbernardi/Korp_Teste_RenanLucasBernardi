import { Routes } from '@angular/router';

export const routes: Routes = [
  { path: '', redirectTo: 'produtos', pathMatch: 'full' },
  {
    path: 'produtos',
    loadComponent: () => import('./produtos/produtos-page').then(m => m.ProdutosPage)
  },
  {
    path: 'notas',
    loadComponent: () => import('./notas/notas-page').then(m => m.NotasPage)
  }
];