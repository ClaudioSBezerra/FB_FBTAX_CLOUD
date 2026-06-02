package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"fb_cloud/services"
)

// OFXUploadHandler trata POST /api/financeiro/ofx/upload
// Aceita multipart/form-data com campo "file" (.ofx) e "conta_id" opcional.
// Se conta_id estiver vazio, tenta auto-detectar via BANKID+ACCTID do arquivo.
// Retorna 409 com payload { detected: { bankid, acctid, branchid } } se a conta não for encontrada.
func OFXUploadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		// Limitar a 5 MB — ParseMultipartForm trunca e retorna erro se exceder
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			http.Error(w, "arquivo muito grande ou multipart inválido", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "campo 'file' obrigatório", http.StatusBadRequest)
			return
		}
		defer file.Close()

		contaID := r.FormValue("conta_id")

		result, ingestErr := services.IngestOFX(db, contaID, file)
		if ingestErr != nil {
			var errNaoDetectada *services.ErrContaNaoDetectada
			if errors.As(ingestErr, &errNaoDetectada) {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"detected": errNaoDetectada.Detected,
				})
				return
			}
			http.Error(w, ingestErr.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(result)
	}
}
