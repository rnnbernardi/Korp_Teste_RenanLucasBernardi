export interface ItemNota {
  codigo: string;
  descricao: string;
  quantidade: number;
}

export interface NotaFiscal {
  id: number;
  numero: number;
  status: string;
  itens: ItemNota[];
}