package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

type Produto struct {
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
	Saldo     int    `json:"saldo"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite", "estoque.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := migrar(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /produtos", handleCriarProduto)
	mux.HandleFunc("GET /produtos", handleListarProdutos)
	mux.HandleFunc("POST /baixas", handleBaixas)
	mux.HandleFunc("PUT /produtos/{codigo}", handleAtualizarProduto)
	handler := recoverPanic(withCORS(mux))

	log.Println("estoque-service rodando em :8081")
	log.Fatal(http.ListenAndServe(":8081", handler))
}

func migrar() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS produtos (
			codigo    TEXT PRIMARY KEY,
			descricao TEXT NOT NULL,
			saldo     INTEGER NOT NULL
		)
	`)
	return err
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleCriarProduto(w http.ResponseWriter, r *http.Request) {
	var p Produto
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, `{"error":"corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}

	if p.Codigo == "" || p.Descricao == "" {
		http.Error(w, `{"error":"codigo e descricao são obrigatórios"}`, http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		"INSERT INTO produtos (codigo, descricao, saldo) VALUES (?, ?, ?)",
		p.Codigo, p.Descricao, p.Saldo,
	)
	if err != nil {
		http.Error(w, `{"error":"não foi possível salvar o produto (código já existe?)"}`, http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func handleListarProdutos(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT codigo, descricao, saldo FROM produtos ORDER BY codigo")
	if err != nil {
		http.Error(w, `{"error":"erro ao consultar produtos"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	produtos := []Produto{}
	for rows.Next() {
		var p Produto
		if err := rows.Scan(&p.Codigo, &p.Descricao, &p.Saldo); err != nil {
			http.Error(w, `{"error":"erro ao ler produtos"}`, http.StatusInternalServerError)
			return
		}
		produtos = append(produtos, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(produtos)
}
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
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

type ItemBaixa struct {
	Codigo     string `json:"codigo"`
	Quantidade int    `json:"quantidade"`
}

type BaixaRequest struct {
	Itens []ItemBaixa `json:"itens"`
}

func handleBaixas(w http.ResponseWriter, r *http.Request) {
	var req BaixaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	if len(req.Itens) == 0 {
		http.Error(w, `{"error":"nenhum item informado"}`, http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, `{"error":"erro ao iniciar transação"}`, http.StatusInternalServerError)
		return
	}

	for _, item := range req.Itens {
		result, err := tx.Exec(
			"UPDATE produtos SET saldo = saldo - ? WHERE codigo = ? AND saldo >= ?",
			item.Quantidade, item.Codigo, item.Quantidade,
		)
		if err != nil {
			tx.Rollback()
			http.Error(w, `{"error":"erro ao atualizar saldo"}`, http.StatusInternalServerError)
			return
		}

		linhas, _ := result.RowsAffected()
		if linhas == 0 {
			tx.Rollback()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":  "saldo insuficiente ou produto não encontrado",
				"codigo": item.Codigo,
			})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"erro ao confirmar transação"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
func handleAtualizarProduto(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")

	var body struct {
		Descricao string `json:"descricao"`
		Saldo     int    `json:"saldo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	if body.Descricao == "" {
		http.Error(w, `{"error":"descricao é obrigatória"}`, http.StatusBadRequest)
		return
	}

	result, err := db.Exec(
		"UPDATE produtos SET descricao = ?, saldo = ? WHERE codigo = ?",
		body.Descricao, body.Saldo, codigo,
	)
	if err != nil {
		http.Error(w, `{"error":"erro ao atualizar produto"}`, http.StatusInternalServerError)
		return
	}

	linhas, _ := result.RowsAffected()
	if linhas == 0 {
		http.Error(w, `{"error":"produto não encontrado"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Produto{Codigo: codigo, Descricao: body.Descricao, Saldo: body.Saldo})
}
