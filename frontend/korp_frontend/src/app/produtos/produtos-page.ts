import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatTableModule } from '@angular/material/table';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ProdutoService } from '../service/produto.service';
import { Produto } from '../models/produto.model';

@Component({
  selector: 'app-produtos-page',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatTableModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule
  ],
  template: `
    <h1>Produtos</h1>

    <form [formGroup]="form" (ngSubmit)="salvar()" class="form-produto">
      <mat-form-field appearance="outline">
        <mat-label>Código</mat-label>
        <input matInput formControlName="codigo" placeholder="P001">
      </mat-form-field>

      <mat-form-field appearance="outline">
        <mat-label>Descrição</mat-label>
        <input matInput formControlName="descricao" placeholder="Caneta azul">
      </mat-form-field>

      <mat-form-field appearance="outline">
        <mat-label>Saldo</mat-label>
        <input matInput type="number" formControlName="saldo">
      </mat-form-field>

      <button mat-flat-button color="primary" type="submit" [disabled]="form.invalid || salvando()">
        @if (salvando()) {
          <mat-spinner diameter="20"></mat-spinner>
        } @else {
          {{ editando() ? 'Salvar alterações' : 'Cadastrar produto' }}
        }
      </button>

      @if (editando()) {
        <button mat-button type="button" (click)="cancelarEdicao()">Cancelar</button>
      }
    </form>

    @if (carregando()) {
      <mat-spinner diameter="32"></mat-spinner>
    } @else {
      <table mat-table [dataSource]="produtos()" class="tabela-produtos">
        <ng-container matColumnDef="codigo">
          <th mat-header-cell *matHeaderCellDef>Código</th>
          <td mat-cell *matCellDef="let p">{{ p.codigo }}</td>
        </ng-container>

        <ng-container matColumnDef="descricao">
          <th mat-header-cell *matHeaderCellDef>Descrição</th>
          <td mat-cell *matCellDef="let p">{{ p.descricao }}</td>
        </ng-container>

        <ng-container matColumnDef="saldo">
          <th mat-header-cell *matHeaderCellDef>Saldo</th>
          <td mat-cell *matCellDef="let p">{{ p.saldo }}</td>
        </ng-container>

        <ng-container matColumnDef="acoes">
          <th mat-header-cell *matHeaderCellDef></th>
          <td mat-cell *matCellDef="let p">
            <button mat-button color="primary" (click)="iniciarEdicao(p)">Editar</button>
          </td>
        </ng-container>

        <tr mat-header-row *matHeaderRowDef="colunas"></tr>
        <tr mat-row *matRowDef="let row; columns: colunas;"></tr>
      </table>
    }
  `,
  styles: [`
    .form-produto {
      display: flex;
      gap: 16px;
      align-items: flex-start;
      flex-wrap: wrap;
      margin-bottom: 24px;
    }
    .tabela-produtos {
      width: 100%;
    }
  `]
})
export class ProdutosPage implements OnInit {
  private produtoService = inject(ProdutoService);
  private fb = inject(FormBuilder);
  private snackBar = inject(MatSnackBar);

  produtos = signal<Produto[]>([]);
  colunas = ['codigo', 'descricao', 'saldo', 'acoes'];
  carregando = signal(false);
  salvando = signal(false);
  editando = signal<Produto | null>(null);

  form = this.fb.nonNullable.group({
    codigo: ['', Validators.required],
    descricao: ['', Validators.required],
    saldo: [0, [Validators.required, Validators.min(0)]]
  });

  ngOnInit(): void {
    this.carregarProdutos();
  }

  carregarProdutos(): void {
    this.carregando.set(true);
    this.produtoService.listar().subscribe({
      next: (produtos) => {
        this.produtos.set(produtos);
        this.carregando.set(false);
      },
      error: () => {
        this.carregando.set(false);
        this.snackBar.open('Erro ao carregar produtos. O serviço de Estoque está no ar?', 'Fechar', { duration: 5000 });
      }
    });
  }

  iniciarEdicao(produto: Produto): void {
    this.editando.set(produto);
    this.form.setValue({
      codigo: produto.codigo,
      descricao: produto.descricao,
      saldo: produto.saldo
    });
    this.form.controls.codigo.disable();
  }

  cancelarEdicao(): void {
    this.editando.set(null);
    this.form.reset({ codigo: '', descricao: '', saldo: 0 });
    this.form.controls.codigo.enable();
  }

  salvar(): void {
    if (this.form.invalid) return;

    this.salvando.set(true);
    const valores = this.form.getRawValue();
    const editandoAtual = this.editando();

    const acao = editandoAtual
      ? this.produtoService.atualizar(editandoAtual.codigo, { descricao: valores.descricao, saldo: valores.saldo })
      : this.produtoService.criar(valores);

    acao.subscribe({
      next: () => {
        this.salvando.set(false);
        this.snackBar.open(
          editandoAtual ? 'Produto atualizado com sucesso!' : 'Produto cadastrado com sucesso!',
          'Fechar',
          { duration: 3000 }
        );
        this.cancelarEdicao();
        this.carregarProdutos();
      },
      error: () => {
        this.salvando.set(false);
        this.snackBar.open('Erro ao salvar produto.', 'Fechar', { duration: 5000 });
      }
    });
  }
}