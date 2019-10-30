package restapp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kvanticoss/gdpr-id-mapper/pkg/idmapper"
)

// Router managed HTTP communication with the GDPR mapping logic
type Router struct {
	mapper *idmapper.GdprMapper
}

// New returns a new Router
func New(mapper *idmapper.GdprMapper) *Router {
	return &Router{
		mapper: mapper,
	}
}

// GetUnProctedMuxer adds the routes under the given muxer
func (rtr *Router) GetUnProctedMuxer() *http.ServeMux {
	mux := &http.ServeMux{}
	mux.HandleFunc("/q/", rtr.query)
	mux.HandleFunc("/c/", rtr.clear)
	mux.HandleFunc("/b/", rtr.bulk)

	return mux
}

func (rtr *Router) bulk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "x-text/json-newline")
	encoder := json.NewEncoder(w)
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		key := scanner.Text()
		byteKey := [][]byte{}
		for _, key := range strings.Split(key, "/") {
			if len(key) == 0 {
				continue
			}
			byteKey = append(byteKey, []byte(key))
		}

		if QueriedRecord, err := rtr.mapper.Query(byteKey, getTTLFromQuery(r.URL)); err != nil {
			_ = encoder.Encode(APIEnvelope{
				Status:  "fail",
				Msg:     err.Error(),
				Payload: nil,
			})
		} else {
			PublicRecord := QueriedRecord.PublicVersion()
			PublicRecord.OriginalID = key //Just make it easier for users we don't expose our internal id
			_ = encoder.Encode(APIEnvelope{
				Status:  "ok",
				Payload: PublicRecord,
			})
		}
	}
}

func (rtr *Router) query(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/json")
	encoder := json.NewEncoder(w)
	key := r.URL.Path[3:]

	byteKey := [][]byte{}
	for _, key := range strings.Split(key, "/") {
		if len(key) == 0 {
			continue
		}
		byteKey = append(byteKey, []byte(key))
	}

	if QueriedRecord, err := rtr.mapper.Query(byteKey, getTTLFromQuery(r.URL)); err != nil {
		_ = encoder.Encode(APIEnvelope{
			Status:  "fail",
			Msg:     err.Error(),
			Payload: nil,
		})
	} else {
		PublicRecord := QueriedRecord.PublicVersion()
		PublicRecord.OriginalID = key //Just make it easier for users we don't expose our internal id
		_ = encoder.Encode(APIEnvelope{
			Status:  "ok",
			Payload: PublicRecord,
		})
	}
}

func (rtr *Router) clear(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/json")
	encoder := json.NewEncoder(w)

	key := r.URL.Path[3:]

	byteKey := [][]byte{}
	for _, key := range strings.Split(key, "/") {
		byteKey = append(byteKey, []byte(key))
	}
	if recsRemove, err := rtr.mapper.ClearPrefix(byteKey); err != nil {
		_ = encoder.Encode(APIEnvelope{
			Status:  "fail",
			Msg:     err.Error(),
			Payload: nil,
		})
	} else {
		_ = encoder.Encode(APIEnvelope{
			Status:  "ok",
			Msg:     fmt.Sprintf("Removed all(%d) records starting with %s", recsRemove, key),
			Payload: nil,
		})
	}
}

func getTTLFromQuery(u *url.URL) *time.Duration {
	ttlDuration := u.Query().Get("ttl")
	if ttlDuration != "" {
		if ttlDur, convErr := time.ParseDuration(ttlDuration); convErr == nil {
			return &ttlDur
		}
	}
	return nil
}
