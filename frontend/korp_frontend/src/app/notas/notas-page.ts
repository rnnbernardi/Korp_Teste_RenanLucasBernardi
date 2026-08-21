import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { finalize } from 'rxjs';
import { MatTableModule } from '@angular/material/table';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { NotaService } from '../service/nota.service';
import { ProdutoService } from '../service/produto.service';
import { NotaFiscal, ItemNota } from '../models/nota.model';
import { Produto } from '../models/produto.model';

@Component({
  selector: 'app-notas-page',
  imports: [
    CommonModule,
    FormsModule,
    MatTableModule,
    MatFormFieldModule,
    MatSelectModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule
  ],
  template: `
    <h1>Notas Fiscais</h1>

    <section class="nova-nota">
      <h2>Nova nota</h2>

      <div class="linha-item">
        <mat-form-field appearance="outline">
          <mat-label>Produto</mat-label>
          <mat-select [(ngModel)]="produtoSelecionado">
            @for (p of produtos(); track p.codigo) {
              <mat-option [value]="p.codigo">{{ p.codigo }} — {{ p.descricao }} (saldo: {{ p.saldo }})</mat-option>
            }
          </mat-select>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>Quantidade</mat-label>
          <input matInput type="number" [(ngModel)]="quantidadeSelecionada" min="1">
        </mat-form-field>

        <button mat-stroked-button (click)="adicionarItem()">Adicionar item</button>
      </div>

      @if (itensNovaNota().length > 0) {
        <table mat-table [dataSource]="itensNovaNota()" class="tabela-itens">
          <ng-container matColumnDef="codigo">
            <th mat-header-cell *matHeaderCellDef>Código</th>
            <td mat-cell *matCellDef="let it">{{ it.codigo }}</td>
          </ng-container>
          <ng-container matColumnDef="descricao">
            <th mat-header-cell *matHeaderCellDef>Descrição</th>
            <td mat-cell *matCellDef="let it">{{ it.descricao }}</td>
          </ng-container>
          <ng-container matColumnDef="quantidade">
            <th mat-header-cell *matHeaderCellDef>Quantidade</th>
            <td mat-cell *matCellDef="let it">{{ it.quantidade }}</td>
          </ng-container>
          <ng-container matColumnDef="acoes">
            <th mat-header-cell *matHeaderCellDef></th>
            <td mat-cell *matCellDef="let it; let i = index">
              <button mat-button color="warn" (click)="removerItem(i)">Remover</button>
            </td>
          </ng-container>
          <tr mat-header-row *matHeaderRowDef="colunasItens"></tr>
          <tr mat-row *matRowDef="let row; columns: colunasItens;"></tr>
        </table>

        <button mat-flat-button color="primary" (click)="criarNota()" [disabled]="criando()">
          @if (criando()) {
            <mat-spinner diameter="20"></mat-spinner>
          } @else {
            Criar nota
          }
        </button>
      }
    </section>

    <section class="lista-notas">
      <h2>Notas emitidas</h2>

      @if (carregando()) {
        <mat-spinner diameter="32"></mat-spinner>
      } @else {
        <table mat-table [dataSource]="notas()" class="tabela-notas">
          <ng-container matColumnDef="numero">
            <th mat-header-cell *matHeaderCellDef>Número</th>
            <td mat-cell *matCellDef="let n">{{ n.numero }}</td>
          </ng-container>

          <ng-container matColumnDef="status">
            <th mat-header-cell *matHeaderCellDef>Status</th>
            <td mat-cell *matCellDef="let n">
              <span class="status-badge"
                    [class.status-aberta]="n.status === 'aberta'"
                    [class.status-fechada]="n.status === 'fechada'">
                {{ n.status === 'aberta' ? 'Aberta' : 'Fechada' }}
              </span>
            </td>
          </ng-container>

          <ng-container matColumnDef="itens">
            <th mat-header-cell *matHeaderCellDef>Itens</th>
            <td mat-cell *matCellDef="let n">{{ n.itens.length }} item(ns)</td>
          </ng-container>

          <ng-container matColumnDef="acoes">
            <th mat-header-cell *matHeaderCellDef></th>
            <td mat-cell *matCellDef="let n">
              <button mat-flat-button color="accent"
                      [disabled]="n.status !== 'aberta' || imprimindoId() === n.id"
                      (click)="imprimir(n)">
                @if (imprimindoId() === n.id) {
                  <mat-spinner diameter="20"></mat-spinner>
                } @else {
                  Imprimir
                }
              </button>
            </td>
          </ng-container>

          <tr mat-header-row *matHeaderRowDef="colunasNotas"></tr>
          <tr mat-row *matRowDef="let row; columns: colunasNotas;"></tr>
        </table>
      }
    </section>
  `,
  styles: [`
    section { margin-bottom: 32px; }
    .linha-item { display: flex; gap: 16px; align-items: flex-start; flex-wrap: wrap; }
    .tabela-itens, .tabela-notas { width: 100%; margin: 16px 0; }
    .status-badge {
      padding: 4px 10px;
      border-radius: 12px;
      font-size: 12px;
      font-weight: 600;
      text-transform: uppercase;
    }
    .status-aberta { background: #e3f2fd; color: #1565c0; }
    .status-fechada { background: #e8f5e9; color: #2e7d32; }
  `]
})
export class NotasPage implements OnInit {
  private notaService = inject(NotaService);
  private produtoService = inject(ProdutoService);
  private snackBar = inject(MatSnackBar);

  produtos = signal<Produto[]>([]);
  notas = signal<NotaFiscal[]>([]);
  itensNovaNota = signal<ItemNota[]>([]);

  produtoSelecionado = '';
  quantidadeSelecionada = 1;

  colunasItens = ['codigo', 'descricao', 'quantidade', 'acoes'];
  colunasNotas = ['numero', 'status', 'itens', 'acoes'];

  carregando = signal(false);
  criando = signal(false);
  imprimindoId = signal<number | null>(null);

  ngOnInit(): void {
    this.produtoService.listar().subscribe({
      next: (produtos) => this.produtos.set(produtos)
    });
    this.carregarNotas();
  }

  carregarNotas(): void {
    this.carregando.set(true);
    this.notaService.listar().subscribe({
      next: (notas) => {
        this.notas.set(notas);
        this.carregando.set(false);
      },
      error: () => {
        this.carregando.set(false);
        this.snackBar.open('Erro ao carregar notas. O serviço de Faturamento está no ar?', 'Fechar', { duration: 5000 });
      }
    });
  }

  adicionarItem(): void {
    if (!this.produtoSelecionado || this.quantidadeSelecionada <= 0) return;

    const produto = this.produtos().find(p => p.codigo === this.produtoSelecionado);
    if (!produto) return;

    this.itensNovaNota.update(itens => [...itens, {
      codigo: produto.codigo,
      descricao: produto.descricao,
      quantidade: this.quantidadeSelecionada
    }]);

    this.produtoSelecionado = '';
    this.quantidadeSelecionada = 1;
  }

  removerItem(index: number): void {
    this.itensNovaNota.update(itens => itens.filter((_, i) => i !== index));
  }

  criarNota(): void {
    if (this.itensNovaNota().length === 0) return;

    this.criando.set(true);
    this.notaService.criar(this.itensNovaNota()).subscribe({
      next: () => {
        this.criando.set(false);
        this.itensNovaNota.set([]);
        this.snackBar.open('Nota criada com sucesso!', 'Fechar', { duration: 3000 });
        this.carregarNotas();
      },
      error: () => {
        this.criando.set(false);
        this.snackBar.open('Erro ao criar nota.', 'Fechar', { duration: 5000 });
      }
    });
  }

  imprimir(nota: NotaFiscal): void {
    this.imprimindoId.set(nota.id);

    this.notaService.imprimir(nota.id)
      .pipe(finalize(() => this.imprimindoId.set(null)))
      .subscribe({
        next: () => {
          this.snackBar.open('Nota impressa com sucesso! Saldo atualizado no estoque.', 'Fechar', { duration: 4000 });
          this.carregarNotas();
        },
        error: (err) => {
          const mensagem = err?.error?.error ?? 'Erro inesperado ao imprimir a nota.';
          this.snackBar.open(mensagem, 'Fechar', { duration: 6000 });
        }
      });
  }
}