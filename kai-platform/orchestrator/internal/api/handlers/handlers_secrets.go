package handlers

import (
	"encoding/json"
	"net/http"

	"kaiplatform.com/orchestrator/internal/api/coordinator"
)

func HandleSecretsList(d *Deps, w http.ResponseWriter, r *http.Request) {
	if d.SecretStore == nil {
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}
	secrets, err := d.SecretStore.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list secrets: %v", err)
		return
	}
	if secrets == nil {
		secrets = []coordinator.SecretMeta{}
	}
	json.NewEncoder(w).Encode(secrets)
}

func HandleSecretsSet(d *Deps, w http.ResponseWriter, r *http.Request) {
	if d.SecretStore == nil {
		writeError(w, http.StatusServiceUnavailable, "secret store not available")
		return
	}

	var input coordinator.SecretInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if input.Value == "" {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}

	if err := d.SecretStore.Set(r.Context(), input.Name, input.Value, input.Description); err != nil {
		writeError(w, http.StatusInternalServerError, "set secret: %v", err)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "name": input.Name})
}

func HandleSecretsDelete(d *Deps, w http.ResponseWriter, r *http.Request, name string) {
	if d.SecretStore == nil {
		writeError(w, http.StatusServiceUnavailable, "secret store not available")
		return
	}

	if err := d.SecretStore.Delete(r.Context(), name); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "name": name})
}
