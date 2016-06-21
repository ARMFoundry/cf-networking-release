package handlers

import (
	"io/ioutil"
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/store"

	"github.com/pivotal-golang/lager"
)

type PoliciesDelete struct {
	Logger      lager.Logger
	Unmarshaler marshal.Unmarshaler
	Store       store.Store
}

func (h *PoliciesDelete) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.Logger.Error("body-read-failed", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body format passed to API should be JSON"}`))
		return
	}

	var payload models.Policies
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		h.Logger.Error("unmarshal-failed", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid values passed to API"}`))
		return
	}

	err = h.Store.Delete(payload.Policies)
	if err != nil {
		h.Logger.Error("store-delete-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database delete failed"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{}`))
	return
}
