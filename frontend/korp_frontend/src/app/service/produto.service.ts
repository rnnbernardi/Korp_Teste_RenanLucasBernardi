import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Produto } from '../models/produto.model';

const ESTOQUE_URL = 'http://localhost:8081';

@Injectable({ providedIn: 'root' })
export class ProdutoService {
  constructor(private http: HttpClient) {}

  listar(): Observable<Produto[]> {
    return this.http.get<Produto[]>(`${ESTOQUE_URL}/produtos`);
  }

  criar(produto: Produto): Observable<Produto> {
    return this.http.post<Produto>(`${ESTOQUE_URL}/produtos`, produto);
  }
    atualizar(codigo: string, dados: { descricao: string; saldo: number }): Observable<Produto> {
    return this.http.put<Produto>(`${ESTOQUE_URL}/produtos/${codigo}`, dados);
  }
}