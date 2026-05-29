// /api/sites — tenant-owned site CRUD. Tenants see only their own sites;
// admins see all. Creating a site emits both nginx and apache vhosts.
package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/DreamyMonk/hostQ-cloud/backend/internal/middleware"
	"github.com/DreamyMonk/hostQ-cloud/backend/internal/sites"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SiteHandlers struct {
	Pool   *pgxpool.Pool
	Vhost  *sites.VhostWriter
}

var domainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

type siteRow struct {
	ID         uuid.UUID `json:"id"`
	Domain     string    `json:"domain"`
	Docroot    string    `json:"docroot"`
	PHPVersion string    `json:"phpVersion"`
	OwnerID    uuid.UUID `json:"ownerId"`
	Suspended  bool      `json:"suspended"`
	CreatedAt  string    `json:"createdAt"`
}

func (h *SiteHandlers) List(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r)
	role := middleware.Role(r)
	var rows []siteRow
	var (
		sql  string
		args []any
	)
	if role == "admin" || role == "superadmin" {
		sql = `SELECT id, domain, docroot, php_version, owner_id, suspended, to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSZ')
		       FROM sites ORDER BY created_at DESC`
	} else {
		sql = `SELECT id, domain, docroot, php_version, owner_id, suspended, to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSZ')
		       FROM sites WHERE owner_id = $1 ORDER BY created_at DESC`
		args = append(args, uid)
	}
	cursor, err := h.Pool.Query(r.Context(), sql, args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cursor.Close()
	for cursor.Next() {
		var s siteRow
		if err := cursor.Scan(&s.ID, &s.Domain, &s.Docroot, &s.PHPVersion, &s.OwnerID, &s.Suspended, &s.CreatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		rows = append(rows, s)
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": rows})
}

type createSiteReq struct {
	Domain     string `json:"domain"`
	PHPVersion string `json:"phpVersion"`
}

func (h *SiteHandlers) Create(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r)
	role := middleware.Role(r)

	var req createSiteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
	req.Domain = strings.TrimPrefix(req.Domain, "www.")
	if !domainRe.MatchString(req.Domain) {
		writeErr(w, http.StatusBadRequest, "invalid domain")
		return
	}
	if req.PHPVersion == "" {
		req.PHPVersion = "8.3"
	}

	// Lookup tenant_id for the owner.
	var tenantID uuid.UUID
	if err := h.Pool.QueryRow(r.Context(), `SELECT tenant_id FROM users WHERE id = $1`, uid).Scan(&tenantID); err != nil {
		writeErr(w, http.StatusInternalServerError, "tenant lookup")
		return
	}

	docroot := h.Vhost.DocrootFor(req.Domain)
	nginxPath, apachePath, err := h.Vhost.Write(req.Domain, docroot, req.PHPVersion)
	if err != nil {
		writeAudit(r, h.Pool, uid, role, "site.create", req.Domain, "failure")
		writeErr(w, http.StatusInternalServerError, "vhost: "+err.Error())
		return
	}

	var id uuid.UUID
	err = h.Pool.QueryRow(r.Context(), `
		INSERT INTO sites (owner_id, tenant_id, domain, docroot, php_version, vhost_path_nginx, vhost_path_apache)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, uid, tenantID, req.Domain, docroot, req.PHPVersion, nginxPath, apachePath).Scan(&id)
	if err != nil {
		writeAudit(r, h.Pool, uid, role, "site.create", req.Domain, "failure")
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	// Also insert primary domain row.
	_, _ = h.Pool.Exec(r.Context(), `INSERT INTO domains (site_id, fqdn, is_primary) VALUES ($1, $2, TRUE)`, id, req.Domain)
	writeAudit(r, h.Pool, uid, role, "site.create", req.Domain, "success")
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "domain": req.Domain})
}

func (h *SiteHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r)
	role := middleware.Role(r)
	id, err := uuid.Parse(chiParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	var (
		domain, owner string
	)
	err = h.Pool.QueryRow(r.Context(), `SELECT domain, owner_id::text FROM sites WHERE id = $1`, id).Scan(&domain, &owner)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if owner != uid.String() && role != "admin" && role != "superadmin" {
		writeErr(w, http.StatusForbidden, "not owner")
		return
	}
	_ = h.Vhost.Remove(domain)
	_, err = h.Pool.Exec(r.Context(), `DELETE FROM sites WHERE id = $1`, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeAudit(r, h.Pool, uid, role, "site.delete", domain, "success")
	w.WriteHeader(http.StatusNoContent)
}
