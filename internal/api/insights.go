package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"expense-tracker/internal/insights"
)

func (s *Server) handleInsights(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "month"
	}

	now := time.Now()
	year := now.Year()
	if v := r.URL.Query().Get("year"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "invalid year")
			return
		}
		year = n
	}

	switch view {
	case "month":
		month := int(now.Month())
		if v := r.URL.Query().Get("month"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 12 {
				writeError(w, http.StatusBadRequest, "invalid month (1-12)")
				return
			}
			month = n
		}
		ins, err := insights.Month(s.db, year, month, now)
		if err != nil {
			log.Printf("api: insights month: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, ins)
	case "year":
		ins, err := insights.Year(s.db, year, now)
		if err != nil {
			log.Printf("api: insights year: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, ins)
	default:
		writeError(w, http.StatusBadRequest, "invalid view (must be 'month' or 'year')")
	}
}
