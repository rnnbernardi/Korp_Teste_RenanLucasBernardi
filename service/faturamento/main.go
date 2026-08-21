package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite", "faturamento.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := migrar(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /notas", handleCriarNota)
	mux.HandleFunc("GET /notas", handleListarNotas)
	mux.HandleFunc("POST /notas/{id}/imprimir", handleImprimirNota)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	handler := recoverPanic(withCORS(mux))

	log.Println("faturamento-service rodando em :8082")
	log.Fatal(http.ListenAndServe(":8082", handler))
}

type ItemNota struct {
	Codigo     string `json:"codigo"`
	Descricao  string `json:"descricao"`
	Quantidade int    `json:"quantidade"`
}

type NotaFiscal struct {
	ID     int64      `json:"id"`
	Numero int64      `json:"numero"`
	Status string     `json:"status"`
	Itens  []ItemNota `json:"itens"`
}

func migrar() error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notas_fiscais (
			id     INTEGER PRIMARY KEY AUTOINCREMENT,
			status TEXT NOT NULL DEFAULT 'aberta'
		)
	`); err != nil {
		return err
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS itens_nota (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			nota_id        INTEGER NOT NULL,
			codigo_produto TEXT NOT NULL,
			descricao      TEXT,
			quantidade     INTEGER NOT NULL,
			FOREIGN KEY (nota_id) REFERENCES notas_fiscais(id)
		)
	`)
	return err
}
func handleCriarNota(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Itens []ItemNota `json:"itens"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	if len(body.Itens) == 0 {
		http.Error(w, `{"error":"a nota precisa ter ao menos um item"}`, http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, `{"error":"erro ao iniciar transação"}`, http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec("INSERT INTO notas_fiscais (status) VALUES ('aberta')")
	if err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"erro ao criar nota"}`, http.StatusInternalServerError)
		return
	}
	notaID, _ := result.LastInsertId()

	for _, item := range body.Itens {
		_, err := tx.Exec(
			"INSERT INTO itens_nota (nota_id, codigo_produto, descricao, quantidade) VALUES (?, ?, ?, ?)",
			notaID, item.Codigo, item.Descricao, item.Quantidade,
		)
		if err != nil {
			tx.Rollback()
			http.Error(w, `{"error":"erro ao salvar item da nota"}`, http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"erro ao confirmar nota"}`, http.StatusInternalServerError)
		return
	}

	nota := NotaFiscal{ID: notaID, Numero: notaID, Status: "aberta", Itens: body.Itens}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(nota)
}

func handleListarNotas(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, status FROM notas_fiscais ORDER BY id DESC")
	if err != nil {
		http.Error(w, `{"error":"erro ao consultar notas"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notas := []NotaFiscal{}
	for rows.Next() {
		var n NotaFiscal
		if err := rows.Scan(&n.ID, &n.Status); err != nil {
			http.Error(w, `{"error":"erro ao ler notas"}`, http.StatusInternalServerError)
			return
		}
		n.Numero = n.ID

		itemRows, err := db.Query(
			"SELECT codigo_produto, descricao, quantidade FROM itens_nota WHERE nota_id = ?", n.ID,
		)
		if err != nil {
			http.Error(w, `{"error":"erro ao consultar itens da nota"}`, http.StatusInternalServerError)
			return
		}
		n.Itens = []ItemNota{}
		for itemRows.Next() {
			var it ItemNota
			itemRows.Scan(&it.Codigo, &it.Descricao, &it.Quantidade)
			n.Itens = append(n.Itens, it)
		}
		itemRows.Close()

		notas = append(notas, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notas)
}

var estoqueURL = "http://localhost:8081" // endereço do serviço de Estoque

type itemBaixaReq struct {
	Codigo     string `json:"codigo"`
	Quantidade int    `json:"quantidade"`
}

type baixaRequest struct {
	Itens []itemBaixaReq `json:"itens"`
}

func handleImprimirNota(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var nota NotaFiscal
	err := db.QueryRow("SELECT id, status FROM notas_fiscais WHERE id = ?", id).Scan(&nota.ID, &nota.Status)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"nota não encontrada"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"erro ao buscar nota"}`, http.StatusInternalServerError)
		return
	}
	nota.Numero = nota.ID

	if nota.Status != "aberta" {
		http.Error(w, `{"error":"a nota não está aberta, não pode ser impressa"}`, http.StatusConflict)
		return
	}

	rows, err := db.Query("SELECT codigo_produto, descricao, quantidade FROM itens_nota WHERE nota_id = ?", nota.ID)
	if err != nil {
		http.Error(w, `{"error":"erro ao buscar itens da nota"}`, http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var it ItemNota
		rows.Scan(&it.Codigo, &it.Descricao, &it.Quantidade)
		nota.Itens = append(nota.Itens, it)
	}
	rows.Close()

	payloadItens := make([]itemBaixaReq, len(nota.Itens))
	for i, it := range nota.Itens {
		payloadItens[i] = itemBaixaReq{Codigo: it.Codigo, Quantidade: it.Quantidade}
	}
	payload, _ := json.Marshal(baixaRequest{Itens: payloadItens})

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, estoqueURL+"/baixas", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Estoque fora do ar, ou não respondeu a tempo — é aqui que o requisito de falha entra em ação
		log.Printf("falha ao chamar o Estoque: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "serviço de estoque indisponível no momento, tente novamente em instantes",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		var estoqueErr map[string]string
		json.NewDecoder(resp.Body).Decode(&estoqueErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "não foi possível dar baixa no estoque: " + estoqueErr["error"],
		})
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, `{"error":"erro inesperado ao consultar o estoque"}`, http.StatusInternalServerError)
		return
	}

	if _, err := db.Exec("UPDATE notas_fiscais SET status = 'fechada' WHERE id = ?", nota.ID); err != nil {
		http.Error(w, `{"error":"baixa feita no estoque, mas erro ao atualizar status da nota"}`, http.StatusInternalServerError)
		return
	}
	nota.Status = "fechada"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nota)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recuperado: %v", err)
				http.Error(w, `{"error":"erro interno do servidor"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
