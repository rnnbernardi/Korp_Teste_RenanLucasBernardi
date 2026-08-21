import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { NotaFiscal, ItemNota } from '../models/nota.model';

const FATURAMENTO_URL = 'http://localhost:8082';

@Injectable({ providedIn: 'root' })
export class NotaService {
  constructor(private http: HttpClient) {}

  listar(): Observable<NotaFiscal[]> {
    return this.http.get<NotaFiscal[]>(`${FATURAMENTO_URL}/notas`);
  }

  criar(itens: ItemNota[]): Observable<NotaFiscal> {
    return this.http.post<NotaFiscal>(`${FATURAMENTO_URL}/notas`, { itens });
  }

  imprimir(id: number): Observable<NotaFiscal> {
    return this.http.post<NotaFiscal>(`${FATURAMENTO_URL}/notas/${id}/imprimir`, {});
  }
}