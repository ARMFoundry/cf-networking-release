package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

type WhoAmIHandler struct {
	Logger    lager.Logger
	Marshaler marshal.Marshaler
}

type WhoAmIResponse struct {
	UserName string `json:"user_name"`
}

func (h *WhoAmIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	whoAmIResponse := WhoAmIResponse{
		UserName: tokenData.UserName,
	}
	responseJSON, err := h.Marshaler.Marshal(whoAmIResponse)
	if err != nil {
		h.Logger.Error("marshal-response", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	return
}
