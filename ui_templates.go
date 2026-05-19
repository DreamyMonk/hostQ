package main

import "html/template"

// icon returns a small inline SVG for the given Lucide icon name.
// We embed a curated subset to avoid an external CDN call and stay within CSP.
func icon(name string) template.HTML {
	if svg, ok := lucideIcons[name]; ok {
		return template.HTML(svg)
	}
	return template.HTML(lucideIcons["circle"])
}

// Lucide-licensed SVG paths (ISC). 18x18, stroke=currentColor, strokeWidth=2.
var lucideIcons = map[string]string{
	"layout":     svg(`<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>`),
	"globe":      svg(`<circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>`),
	"folder":     svg(`<path d="M4 5a2 2 0 0 1 2-2h4l2 3h6a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2z"/>`),
	"folderOpen": svg(`<path d="M3 7a2 2 0 0 1 2-2h4l2 3h8a2 2 0 0 1 2 2v1H5a2 2 0 0 0-2 2z"/><path d="M3 13l1.5 6a1 1 0 0 0 1 .8h13a1 1 0 0 0 1-.8L21 13z"/>`),
	"file":       svg(`<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/>`),
	"database":   svg(`<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v6c0 1.7 4 3 9 3s9-1.3 9-3V5"/><path d="M3 11v6c0 1.7 4 3 9 3s9-1.3 9-3v-6"/>`),
	"wordpress":  svg(`<circle cx="12" cy="12" r="10"/><path d="M4 9h6l3 8 3-8h3M7 9l4 11"/>`),
	"shield":     svg(`<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>`),
	"cpu":        svg(`<rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M1 9h3M1 15h3M20 9h3M20 15h3"/>`),
	"clock":      svg(`<circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>`),
	"settings":   svg(`<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9c.36.16.65.46.81.79.16.33.21.7.13 1.06"/>`),
	"server":     svg(`<rect x="2" y="3" width="20" height="8" rx="2"/><rect x="2" y="13" width="20" height="8" rx="2"/><path d="M6 7h.01M6 17h.01"/>`),
	"archive":    svg(`<rect x="2" y="3" width="20" height="5" rx="1"/><path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8"/><path d="M10 12h4"/>`),
	"users":      svg(`<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>`),
	"logout":     svg(`<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><path d="M16 17l5-5-5-5"/><path d="M21 12H9"/>`),
	"plus":       svg(`<path d="M12 5v14M5 12h14"/>`),
	"trash":      svg(`<polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>`),
	"edit":       svg(`<path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 1 1 3 3L7 19l-4 1 1-4z"/>`),
	"copy":       svg(`<rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>`),
	"move":       svg(`<path d="M5 9l-3 3 3 3M9 5l3-3 3 3M15 19l-3 3-3-3M19 9l3 3-3 3M2 12h20M12 2v20"/>`),
	"download":   svg(`<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><path d="M12 15V3"/>`),
	"upload":     svg(`<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><path d="M12 3v12"/>`),
	"refresh":    svg(`<polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>`),
	"check":      svg(`<polyline points="20 6 9 17 4 12"/>`),
	"x":          svg(`<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>`),
	"chevronUp":  svg(`<polyline points="18 15 12 9 6 15"/>`),
	"chevron":    svg(`<polyline points="9 18 15 12 9 6"/>`),
	"home":       svg(`<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2h-4v-7h-6v7H5a2 2 0 0 1-2-2z"/>`),
	"key":        svg(`<path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>`),
	"info":       svg(`<circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/>`),
	"alert":      svg(`<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><path d="M12 9v4M12 17h.01"/>`),
	"power":      svg(`<path d="M18.36 6.64a9 9 0 1 1-12.73 0"/><line x1="12" y1="2" x2="12" y2="12"/>`),
	"play":       svg(`<polygon points="5 3 19 12 5 21 5 3"/>`),
	"stop":       svg(`<rect x="6" y="6" width="12" height="12" rx="1"/>`),
	"terminal":   svg(`<polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>`),
	"box":        svg(`<path d="M21 16V8l-9-5-9 5v8l9 5z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/>`),
	"hardDrive":  svg(`<line x1="22" y1="12" x2="2" y2="12"/><path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/><line x1="6" y1="16" x2="6.01" y2="16"/>`),
	"activity":   svg(`<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>`),
	"circle":     svg(`<circle cx="12" cy="12" r="9"/>`),
	"logo":       svg(`<rect x="3" y="3" width="18" height="18" rx="4"/><path d="M9 9h6v6H9z"/><path d="M12 15v3"/>`),
}

func svg(inner string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + inner + `</svg>`
}

const layoutTemplate = `
{{define "layout"}}<!doctype html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>hostQ — {{.Title}}</title>
<style>
:root{
  --bg:#0b1220;--panel:#0f172a;--panel-2:#1e293b;--text:#e2e8f0;--muted:#94a3b8;--line:#1e293b;
  --card:#ffffff;--card-line:#e5e7eb;--ink:#0f172a;--ink-muted:#64748b;
  --brand:#3b82f6;--brand-2:#2563eb;--accent:#06b6d4;
  --ok:#16a34a;--bad:#dc2626;--warn:#d97706;
  --radius:10px;--shadow:0 1px 2px rgba(15,23,42,.04),0 4px 14px rgba(15,23,42,.06);
}
*{box-sizing:border-box}
html,body{margin:0;padding:0;background:#f1f5f9;color:var(--ink);font-family:Inter,ui-sans-serif,system-ui,Segoe UI,Roboto,sans-serif;font-size:14px;line-height:1.5}
a{color:inherit;text-decoration:none}
button{font-family:inherit;font-size:inherit}
svg{flex:none;vertical-align:middle}
.mono{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:12.5px}
.muted{color:var(--ink-muted)}
.ok{color:var(--ok)}.bad{color:var(--bad)}.warn{color:var(--warn)}

/* shell */
.shell{display:grid;grid-template-columns:248px 1fr;min-height:100vh}
aside.side{background:var(--bg);color:var(--text);padding:18px 12px;position:sticky;top:0;height:100vh;overflow-y:auto;border-right:1px solid #0b1325}
.brand{display:flex;align-items:center;gap:10px;padding:8px 10px 18px;font-weight:800;font-size:18px;color:#fff}
.brand .mark{width:34px;height:34px;border-radius:9px;background:linear-gradient(135deg,#3b82f6,#06b6d4);display:grid;place-items:center;color:#fff;box-shadow:0 6px 20px rgba(59,130,246,.35)}
.brand .mark svg{color:#fff}
.brand small{display:block;font-size:11px;font-weight:600;color:var(--muted);letter-spacing:.06em;text-transform:uppercase}
.navgroup{margin:10px 0 4px;padding:0 12px;font-size:11px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:#64748b}
.nav a{display:flex;align-items:center;gap:10px;padding:8px 12px;border-radius:8px;color:#cbd5e1;font-weight:600;margin-bottom:2px;transition:background .15s,color .15s}
.nav a:hover{background:#0b1933;color:#fff}
.nav a.active{background:linear-gradient(90deg,rgba(59,130,246,.22),transparent);color:#fff;box-shadow:inset 2px 0 0 var(--brand)}
.nav a svg{opacity:.9}
.side-foot{margin-top:auto;padding:14px 10px 4px;border-top:1px solid #0b1933;font-size:12px;color:#94a3b8}
.side-foot .row{display:flex;align-items:center;justify-content:space-between;gap:8px}

/* topbar */
main.main{min-width:0}
.topbar{position:sticky;top:0;z-index:10;background:#fff;border-bottom:1px solid var(--card-line);height:60px;display:flex;align-items:center;justify-content:space-between;padding:0 22px;box-shadow:0 1px 0 rgba(15,23,42,.02)}
.topbar h1{margin:0;font-size:16px;font-weight:700;display:flex;align-items:center;gap:10px}
.topbar .right{display:flex;align-items:center;gap:8px;color:var(--ink-muted);font-size:12.5px}
.topbar .right .chip{display:inline-flex;align-items:center;gap:6px;padding:5px 10px;border-radius:999px;background:#f1f5f9;border:1px solid var(--card-line);color:#334155;font-weight:600}
.content{padding:22px}

/* cards */
.card{background:var(--card);border:1px solid var(--card-line);border-radius:var(--radius);padding:18px;margin-bottom:14px;box-shadow:var(--shadow)}
.card h2{margin:0 0 4px;font-size:15px;font-weight:700}
.card h3{margin:0 0 8px;font-size:13px;font-weight:700;color:#475569;text-transform:uppercase;letter-spacing:.06em}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:14px}
.grid-2{display:grid;grid-template-columns:repeat(auto-fit,minmax(360px,1fr));gap:14px}

/* stat cards */
.stat{display:flex;flex-direction:column;gap:6px;background:#fff;border:1px solid var(--card-line);border-radius:var(--radius);padding:16px;box-shadow:var(--shadow)}
.stat .label{display:flex;align-items:center;gap:8px;color:var(--ink-muted);font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:.06em}
.stat .val{font-size:24px;font-weight:800;color:#0f172a}
.stat .sub{font-size:12px;color:var(--ink-muted)}
.bar{height:6px;border-radius:99px;background:#e2e8f0;overflow:hidden;margin-top:4px}
.bar > div{height:100%;background:linear-gradient(90deg,#3b82f6,#06b6d4)}

/* buttons */
.btn{display:inline-flex;align-items:center;gap:6px;border:1px solid var(--card-line);background:#fff;color:#0f172a;border-radius:8px;padding:8px 12px;font-weight:600;cursor:pointer;transition:background .15s,border-color .15s,transform .05s}
.btn:hover{background:#f8fafc;border-color:#cbd5e1}
.btn:active{transform:translateY(1px)}
.btn.primary{background:var(--brand-2);border-color:var(--brand-2);color:#fff}
.btn.primary:hover{background:#1d4ed8;border-color:#1d4ed8}
.btn.ghost{background:transparent}
.btn.danger{color:#b91c1c;border-color:#fecaca;background:#fff5f5}
.btn.danger:hover{background:#fee2e2}
.btn.mini{padding:4px 8px;font-size:12px;border-radius:6px}
.btn.icon{padding:7px;border-radius:8px}
.actions{display:flex;gap:6px;flex-wrap:wrap}

/* inputs */
.input,select.input,textarea.input{width:100%;border:1px solid var(--card-line);background:#fff;border-radius:8px;padding:9px 12px;font-size:14px;color:#0f172a;outline:none;transition:border-color .15s,box-shadow .15s}
.input:focus{border-color:#93c5fd;box-shadow:0 0 0 3px rgba(59,130,246,.15)}
.field{display:flex;flex-direction:column;gap:6px;margin-bottom:10px}
.field label{font-size:12px;font-weight:700;color:#334155}
.row{display:flex;gap:10px;flex-wrap:wrap}
.row > *{flex:1;min-width:160px}

/* table */
table{width:100%;border-collapse:separate;border-spacing:0;background:#fff;border:1px solid var(--card-line);border-radius:var(--radius);overflow:hidden}
th,td{padding:11px 12px;text-align:left;font-size:13px;border-bottom:1px solid var(--card-line);vertical-align:middle}
th{font-size:11px;font-weight:700;color:#64748b;text-transform:uppercase;letter-spacing:.06em;background:#f8fafc}
tbody tr:last-child td{border-bottom:none}
tbody tr:hover{background:#f8fafc}

/* badges */
.badge{display:inline-flex;align-items:center;gap:5px;border:1px solid var(--card-line);border-radius:999px;padding:2px 9px;font-size:11.5px;font-weight:700;background:#fff;color:#475569}
.badge.ok{color:#166534;border-color:#bbf7d0;background:#f0fdf4}
.badge.bad{color:#991b1b;border-color:#fecaca;background:#fef2f2}
.badge.warn{color:#92400e;border-color:#fde68a;background:#fffbeb}
.badge.info{color:#1e40af;border-color:#bfdbfe;background:#eff6ff}

/* page heading */
.page-head{display:flex;justify-content:space-between;align-items:center;margin:0 0 16px;gap:10px;flex-wrap:wrap}
.page-head h1{margin:0;font-size:22px;font-weight:800;letter-spacing:-.01em}
.page-head p{margin:4px 0 0;color:var(--ink-muted)}

/* file manager */
.crumbs{display:flex;align-items:center;gap:4px;flex-wrap:wrap;background:#fff;border:1px solid var(--card-line);border-radius:8px;padding:8px 12px;margin-bottom:12px;font-size:13px}
.crumbs a{padding:3px 6px;border-radius:5px;color:#0f172a;font-weight:600}
.crumbs a:hover{background:#f1f5f9;color:#1d4ed8}
.crumbs .sep{color:#94a3b8;margin:0 2px}
.fm-toolbar{display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin-bottom:10px}
.fm-toolbar form{display:inline-flex}
.fm-toolbar .grow{flex:1}
.file-name{display:flex;align-items:center;gap:8px;font-weight:600}
.file-name .ic{width:28px;height:28px;border-radius:6px;background:#eff6ff;color:#2563eb;display:grid;place-items:center}
.file-name.dir .ic{background:#fef3c7;color:#b45309}
.fm-table tbody tr{cursor:default}
.fm-table .right-col{text-align:right;white-space:nowrap}

/* context menu */
.ctxmenu{position:fixed;z-index:1000;min-width:200px;background:#fff;border:1px solid var(--card-line);border-radius:8px;box-shadow:0 10px 30px rgba(15,23,42,.18);padding:6px;display:none}
.ctxmenu.show{display:block}
.ctxmenu button{width:100%;border:none;background:none;text-align:left;padding:8px 10px;border-radius:6px;display:flex;align-items:center;gap:10px;cursor:pointer;color:#0f172a;font-weight:600}
.ctxmenu button:hover{background:#f1f5f9}
.ctxmenu .sep{height:1px;background:var(--card-line);margin:4px 2px}
.ctxmenu .danger{color:#b91c1c}

/* modal */
.modal-bg{position:fixed;inset:0;background:rgba(15,23,42,.45);z-index:900;display:none;align-items:center;justify-content:center;padding:20px}
.modal-bg.show{display:flex}
.modal{background:#fff;border-radius:12px;padding:20px;width:100%;max-width:480px;box-shadow:0 20px 60px rgba(15,23,42,.25)}
.modal h3{margin:0 0 4px;font-size:16px;font-weight:800}
.modal p.muted{margin:0 0 14px}
.modal .modal-foot{display:flex;gap:8px;justify-content:flex-end;margin-top:6px}

/* login */
.login-wrap{min-height:100vh;display:grid;place-items:center;padding:24px;background:radial-gradient(1200px 600px at 20% 0%,#dbeafe,transparent),radial-gradient(900px 500px at 100% 100%,#cffafe,transparent),#f1f5f9}
.login-card{width:100%;max-width:400px;background:#fff;padding:28px;border-radius:14px;box-shadow:0 12px 40px rgba(15,23,42,.10),0 2px 4px rgba(15,23,42,.04);border:1px solid var(--card-line)}
.login-card h1{font-size:22px;margin:6px 0 2px;display:flex;align-items:center;gap:10px}
.login-card .brand-pill{width:38px;height:38px;border-radius:10px;background:linear-gradient(135deg,#3b82f6,#06b6d4);color:#fff;display:grid;place-items:center}

/* alerts */
.alert{padding:10px 12px;border-radius:8px;border:1px solid;display:flex;align-items:center;gap:8px;font-weight:600;margin-bottom:12px}
.alert.ok{color:#166534;border-color:#bbf7d0;background:#f0fdf4}
.alert.bad{color:#991b1b;border-color:#fecaca;background:#fef2f2}
.alert.info{color:#1e40af;border-color:#bfdbfe;background:#eff6ff}

pre.mono{background:#0b1220;color:#e2e8f0;padding:14px;border-radius:8px;overflow:auto;font-size:12px;white-space:pre-wrap}
hr.sep{border:none;border-top:1px solid var(--card-line);margin:14px 0}

/* tabbed site manager */
.tabs{display:flex;gap:2px;background:#fff;border:1px solid var(--card-line);border-radius:10px;padding:6px;margin-bottom:14px;overflow-x:auto;flex-wrap:wrap;box-shadow:var(--shadow)}
.tabs a{display:inline-flex;align-items:center;gap:6px;padding:8px 14px;border-radius:7px;color:#475569;font-weight:600;font-size:13.5px;white-space:nowrap;transition:background .15s,color .15s}
.tabs a:hover{background:#f1f5f9;color:#0f172a}
.tabs a.active{background:linear-gradient(135deg,#2563eb,#06b6d4);color:#fff;box-shadow:0 4px 12px rgba(37,99,235,.25)}
.tabs a.active svg{color:#fff}

@media (max-width:880px){
  .shell{grid-template-columns:1fr}
  aside.side{position:relative;height:auto}
}
</style>
</head><body>
{{if eq .View "login"}}
  {{template "login" .}}
{{else}}
<div class="shell">
  <aside class="side">
    <div class="brand">
      <span class="mark">{{icon "logo"}}</span>
      <span>hostQ<small>control panel</small></span>
    </div>
    <div class="navgroup">Main</div>
    <nav class="nav">
      <a href="/" class="{{if eq .View "dashboard"}}active{{end}}">{{icon "layout"}}<span>Dashboard</span></a>
      <a href="/sites" class="{{if or (eq .View "sites") (eq .View "site")}}active{{end}}">{{icon "globe"}}<span>Sites</span></a>
      <a href="/files?path=/" class="{{if eq .View "files"}}active{{end}}">{{icon "folder"}}<span>File Manager</span></a>
    </nav>
    <div class="navgroup">Server</div>
    <nav class="nav">
      <a href="/services" class="{{if eq .View "services"}}active{{end}}">{{icon "server"}}<span>Services</span></a>
      <a href="/cron" class="{{if eq .View "cron"}}active{{end}}">{{icon "clock"}}<span>Cron</span></a>
      <a href="/php" class="{{if eq .View "php"}}active{{end}}">{{icon "cpu"}}<span>PHP Versions</span></a>
    </nav>
    <div class="navgroup">Advanced</div>
    <nav class="nav">
      <a href="/databases" class="{{if eq .View "databases"}}active{{end}}">{{icon "database"}}<span>All Databases</span></a>
      <a href="/wordpress" class="{{if eq .View "wordpress"}}active{{end}}">{{icon "wordpress"}}<span>All WordPress</span></a>
      <a href="/ssl" class="{{if eq .View "ssl"}}active{{end}}">{{icon "shield"}}<span>All Certificates</span></a>
    </nav>
    <div class="navgroup">Admin</div>
    <nav class="nav">
      <a href="/account" class="{{if eq .View "account"}}active{{end}}">{{icon "users"}}<span>Account</span></a>
      <a href="/audit" class="{{if eq .View "audit"}}active{{end}}">{{icon "activity"}}<span>Audit Log</span></a>
      <a href="/logout">{{icon "logout"}}<span>Sign out</span></a>
    </nav>
    <div class="side-foot">
      <div class="row"><span>{{now.Format "Mon Jan 02"}}</span><span>{{now.Format "15:04"}}</span></div>
    </div>
  </aside>
  <main class="main">
    <header class="topbar">
      <h1>{{icon "chevron"}}{{.Title}}</h1>
      <div class="right">
        <span class="chip">{{icon "circle"}} healthy</span>
        <span class="chip">{{icon "server"}} single server</span>
      </div>
    </header>
    <div class="content">
      {{if eq .View "dashboard"}}{{template "dashboard" .}}
      {{else if eq .View "sites"}}{{template "sites" .}}
      {{else if eq .View "site"}}{{template "site" .}}
      {{else if eq .View "backups"}}{{template "backups" .}}
      {{else if eq .View "wordpress"}}{{template "wordpress" .}}
      {{else if eq .View "files"}}{{template "files" .}}
      {{else if eq .View "databases"}}{{template "databases" .}}
      {{else if eq .View "php"}}{{template "php" .}}
      {{else if eq .View "ssl"}}{{template "ssl" .}}
      {{else if eq .View "services"}}{{template "services" .}}
      {{else if eq .View "cron"}}{{template "cron" .}}
      {{else if eq .View "account"}}{{template "account" .}}
      {{else if eq .View "audit"}}{{template "audit" .}}
      {{end}}
    </div>
  </main>
</div>
<script>
// Confirm on data-confirm forms
document.addEventListener('submit',function(e){
  var m=(e.submitter&&e.submitter.getAttribute('data-confirm'))||e.target.getAttribute('data-confirm');
  if(m && !confirm(m)){ e.preventDefault(); }
});
// Generic modal helpers
function openModal(id){ var el=document.getElementById(id); if(el){ el.classList.add('show'); } }
function closeModal(id){ var el=document.getElementById(id); if(el){ el.classList.remove('show'); } }
document.addEventListener('click',function(e){
  if(e.target.classList && e.target.classList.contains('modal-bg')){ e.target.classList.remove('show'); }
});
</script>
{{end}}
</body></html>
{{end}}

{{define "login"}}
<div class="login-wrap">
  <div class="login-card">
    <h1><span class="brand-pill">{{icon "logo"}}</span> hostQ</h1>
    <p class="muted">Sign in to your control panel.</p>
    {{if .Error}}<div class="alert bad">{{icon "alert"}} {{.Error}}</div>{{end}}
    <form method="post">
      <div class="field"><label>Username</label><input class="input" name="username" autocomplete="username" autofocus></div>
      <div class="field"><label>Password</label><input class="input" type="password" name="password" autocomplete="current-password"></div>
      <button class="btn primary" style="width:100%;justify-content:center" type="submit">{{icon "key"}} Sign in</button>
    </form>
  </div>
</div>
{{end}}

{{define "dashboard"}}
<div class="page-head">
  <div><h1>Welcome back</h1><p>Overview of your server — {{.Stats.Hostname}} · {{.Stats.CPUCount}} CPU</p></div>
  <span class="badge info">{{icon "clock"}} uptime {{.Stats.Uptime}}</span>
</div>
<div class="grid">
  <div class="stat"><div class="label">{{icon "globe"}} Sites</div><div class="val">{{len .Sites}}</div><div class="sub">Configured vhosts</div></div>
  <div class="stat"><div class="label">{{icon "server"}} Services</div><div class="val">{{len .Services}}</div><div class="sub">Tracked daemons</div></div>
  <div class="stat"><div class="label">{{icon "activity"}} Load avg</div><div class="val">{{.Stats.LoadAvg}}</div><div class="sub">1, 5, 15 min</div></div>
  <div class="stat"><div class="label">{{icon "cpu"}} Memory</div><div class="val">{{.Stats.MemPercent}}%</div><div class="sub">{{.Stats.MemUsed}} / {{.Stats.MemTotal}}</div><div class="bar"><div style="width:{{.Stats.MemPercent}}%"></div></div></div>
  <div class="stat"><div class="label">{{icon "hardDrive"}} Web disk</div><div class="val">{{.Stats.DiskPercent}}%</div><div class="sub">{{.Stats.DiskUsed}} / {{.Stats.DiskTotal}}</div><div class="bar"><div style="width:{{.Stats.DiskPercent}}%"></div></div></div>
</div>
<div class="grid-2" style="margin-top:14px">
  <div class="card">
    <h2>Sites</h2>
    <p class="muted">Recent vhosts. <a href="/sites" style="color:#2563eb;font-weight:700">View all →</a></p>
    <table style="margin-top:10px">
      <thead><tr><th>Domain</th><th>PHP</th><th>Status</th><th></th></tr></thead>
      <tbody>
        {{range .Sites}}<tr>
          <td><strong>{{.Domain}}</strong></td>
          <td><span class="badge">PHP {{.PHPVersion}}</span></td>
          <td>{{if .Enabled}}<span class="badge ok">{{icon "check"}} enabled</span>{{else}}<span class="badge bad">{{icon "x"}} disabled</span>{{end}}</td>
          <td class="right-col"><a class="btn mini" href="/site?domain={{.Domain}}">Open</a></td>
        </tr>{{else}}<tr><td colspan="4" class="muted">No sites yet — <a href="/sites" style="color:#2563eb">add your first</a>.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
  <div class="card">
    <h2>Services</h2>
    <p class="muted">Run/restart from <a href="/services" style="color:#2563eb;font-weight:700">Services →</a></p>
    <table style="margin-top:10px">
      <thead><tr><th>Name</th><th>Status</th></tr></thead>
      <tbody>
        {{range .Services}}<tr>
          <td>{{.Name}} <span class="muted mono">({{.Systemd}})</span></td>
          <td>{{if eq .Status "active"}}<span class="badge ok">{{icon "check"}} active</span>{{else}}<span class="badge bad">{{icon "alert"}} {{.Status}}</span>{{end}}</td>
        </tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
{{end}}

{{define "sites"}}
<div class="page-head">
  <div><h1>Sites</h1><p>Create, enable, and manage your web sites.</p></div>
</div>
{{if .Error}}<div class="alert bad">{{icon "alert"}} {{.Error}}</div>{{end}}
<div class="card">
  <h3>Add new site</h3>
  <form method="post">
    <div class="row">
      <input class="input" name="domain" placeholder="example.com" required>
      <button class="btn primary">{{icon "plus"}} Add site</button>
    </div>
    <p class="muted" style="margin-top:8px">An Nginx vhost and document root will be created. PHP 8.4 by default — change later in PHP Manager.</p>
  </form>
</div>
<div class="card" style="padding:0">
  <table>
    <thead><tr><th>Domain</th><th>Document root</th><th>PHP</th><th>Cache</th><th>SSL</th><th>Status</th><th class="right-col">Action</th></tr></thead>
    <tbody>
      {{range .Sites}}<tr>
        <td><strong>{{.Domain}}</strong></td>
        <td class="mono muted">{{.Root}}</td>
        <td><span class="badge">{{.PHPVersion}}</span></td>
        <td>{{if .Cache}}<span class="badge ok">on</span>{{else}}<span class="badge">off</span>{{end}}</td>
        <td>{{if .SSL}}<span class="badge ok">{{icon "shield"}} on</span>{{else}}<span class="badge">off</span>{{end}}</td>
        <td>{{if .Enabled}}<span class="badge ok">{{icon "check"}} enabled</span>{{else}}<span class="badge bad">{{icon "x"}} disabled</span>{{end}}</td>
        <td class="right-col"><a class="btn mini primary" href="/site?domain={{.Domain}}">Open Manager</a></td>
      </tr>{{else}}<tr><td colspan="7" class="muted">No sites yet. Add one above.</td></tr>{{end}}
    </tbody>
  </table>
</div>
{{end}}

{{define "site"}}
<div class="page-head">
  <div>
    <h1>{{.Site.Domain}}</h1>
    <p class="muted mono">{{.Site.Root}}</p>
  </div>
  <div class="actions">
    {{if .Site.Enabled}}<span class="badge ok">{{icon "check"}} enabled</span>{{else}}<span class="badge bad">disabled</span>{{end}}
    {{if .Site.SSL}}<span class="badge ok">{{icon "shield"}} SSL</span>{{end}}
    {{if .Site.Cache}}<span class="badge info">cache</span>{{end}}
    <span class="badge">PHP {{.Site.PHPVersion}}</span>
    <a class="btn" href="http{{if .Site.SSL}}s{{end}}://{{.Site.Domain}}" target="_blank">{{icon "globe"}} Visit</a>
    <a class="btn" href="/sites">{{icon "chevronUp"}} Back</a>
  </div>
</div>

<div class="tabs">
  <a href="/site?domain={{.Site.Domain}}&tab=overview"  class="{{if eq .Tab "overview"}}active{{end}}">{{icon "layout"}} Overview</a>
  <a href="/site?domain={{.Site.Domain}}&tab=database"  class="{{if eq .Tab "database"}}active{{end}}">{{icon "database"}} Database</a>
  <a href="/site?domain={{.Site.Domain}}&tab=wordpress" class="{{if eq .Tab "wordpress"}}active{{end}}">{{icon "wordpress"}} WordPress</a>
  <a href="/site?domain={{.Site.Domain}}&tab=ssl"       class="{{if eq .Tab "ssl"}}active{{end}}">{{icon "shield"}} SSL</a>
  <a href="/site?domain={{.Site.Domain}}&tab=php"       class="{{if eq .Tab "php"}}active{{end}}">{{icon "cpu"}} PHP</a>
  <a href="/files?path=/{{.Site.Domain}}/htdocs">{{icon "folderOpen"}} Files</a>
  <a href="/site?domain={{.Site.Domain}}&tab=backups"   class="{{if eq .Tab "backups"}}active{{end}}">{{icon "archive"}} Backups</a>
</div>

{{if .Output}}<div class="alert info">{{icon "info"}} {{.Output}}</div>{{end}}

{{if eq .Tab "overview"}}
  <div class="grid">
    <div class="stat"><div class="label">{{icon "cpu"}} PHP version</div><div class="val">{{.Site.PHPVersion}}</div><div class="sub">FastCGI Process Manager</div></div>
    <div class="stat"><div class="label">{{icon "shield"}} SSL</div><div class="val">{{if .Site.SSL}}<span class="ok">on</span>{{else}}<span class="bad">off</span>{{end}}</div><div class="sub">Let's Encrypt</div></div>
    <div class="stat"><div class="label">{{icon "activity"}} Cache</div><div class="val">{{if .Site.Cache}}<span class="ok">on</span>{{else}}off{{end}}</div><div class="sub">Nginx fastcgi cache</div></div>
    <div class="stat"><div class="label">{{icon "globe"}} Status</div><div class="val">{{if .Site.Enabled}}<span class="ok">live</span>{{else}}<span class="bad">disabled</span>{{end}}</div><div class="sub">{{.Site.Domain}}</div></div>
  </div>
  <div class="card">
    <h3>Site controls</h3>
    <div class="actions">
      <form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Site.Domain}}">
        {{if .Site.Enabled}}<button class="btn" name="action" value="disable">{{icon "power"}} Disable site</button>
        {{else}}<button class="btn primary" name="action" value="enable">{{icon "power"}} Enable site</button>{{end}}
      </form>
      <form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Site.Domain}}">
        {{if .Site.Cache}}<button class="btn" name="action" value="cache-off">Cache: off</button>
        {{else}}<button class="btn" name="action" value="cache-on">Cache: on</button>{{end}}
      </form>
      <form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Site.Domain}}">
        <button class="btn" name="action" value="permissions">{{icon "shield"}} Fix permissions</button>
      </form>
      <a class="btn" href="/files?path=/{{.Site.Domain}}/htdocs">{{icon "folderOpen"}} Open files</a>
      <a class="btn" href="/phpmyadmin/" target="_blank">{{icon "database"}} phpMyAdmin</a>
      <form method="post" action="/site-action" data-confirm="Permanently delete this site, its document root, and Nginx vhost?">
        <input type="hidden" name="domain" value="{{.Site.Domain}}">
        <button class="btn danger" name="action" value="delete">{{icon "trash"}} Delete site</button>
      </form>
    </div>
  </div>
{{end}}

{{if eq .Tab "database"}}
  {{if .Created}}<div class="alert ok">{{icon "check"}}
    <div><strong>Database created:</strong> <span class="mono">{{.Created}}</span> · <strong>User:</strong> <span class="mono">{{.User}}</span> · <strong>Password:</strong> <span class="mono">{{.Password}}</span><div class="muted" style="font-weight:500">Save this password — shown only once.</div></div>
  </div>{{end}}
  {{if .DBUser}}<div class="alert ok">{{icon "key"}} <div><strong>DB user:</strong> <span class="mono">{{.DBUser}}</span> · <strong>Password:</strong> <span class="mono">{{.DBPass}}</span>{{if .DBName}} · <strong>Database:</strong> <span class="mono">{{.DBName}}</span>{{end}}<div class="muted" style="font-weight:500">Shown only once.</div></div></div>{{end}}
  <div class="card">
    <h3>Create database for {{.Site.Domain}}</h3>
    <form method="post" action="/databases">
      <input type="hidden" name="site" value="{{.Site.Domain}}">
      <div class="row">
        <div class="field"><label>Suffix (will become <span class="mono">{{.DBPrefix}}_&lt;suffix&gt;</span>)</label><input class="input" name="name" placeholder="main" required></div>
        <button class="btn primary" name="action" value="create" style="align-self:flex-end">{{icon "plus"}} Create database</button>
      </div>
      <p class="muted">A database, matching user, and 24-char password are generated atomically. All databases for this site share the <span class="mono">{{.DBPrefix}}_</span> prefix so the dashboard can scope them per-site.</p>
    </form>
  </div>
  {{range .Databases}}
  <div class="card">
    <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px">
      <h3 style="margin:0">{{icon "database"}} <span class="mono">{{.Name}}</span></h3>
      <div class="actions">
        <a class="btn mini" href="/phpmyadmin/?db={{.Name}}" target="_blank">{{icon "terminal"}} phpMyAdmin</a>
        <form method="post" action="/databases" data-confirm="Drop database {{.Name}}? This deletes all tables and the matching user.">
          <input type="hidden" name="site" value="{{$.Site.Domain}}">
          <input type="hidden" name="target" value="{{.Name}}">
          <button class="btn mini danger" name="action" value="delete">{{icon "trash"}} Drop</button>
        </form>
      </div>
    </div>
    <hr class="sep">
    <h4 class="muted" style="margin:0 0 8px">Users with access</h4>
    <table>
      <thead><tr><th>User</th><th>Host</th><th class="right-col">Actions</th></tr></thead>
      <tbody>
        {{range .Users}}<tr>
          <td class="mono"><strong>{{.Login}}</strong></td>
          <td class="muted mono">{{.Host}}</td>
          <td class="right-col"><div class="actions" style="justify-content:flex-end">
            <form method="post" action="/databases" style="display:inline-flex;gap:6px;flex-wrap:wrap">
              <input type="hidden" name="site" value="{{$.Site.Domain}}">
              <input type="hidden" name="user" value="{{.Login}}">
              <input class="input mini" name="password" type="text" placeholder="new password" minlength="8" required style="width:170px">
              <button class="btn mini" name="action" value="user-password">{{icon "key"}} Reset password</button>
            </form>
            <form method="post" action="/databases" data-confirm="Drop user {{.Login}} and revoke access?">
              <input type="hidden" name="site" value="{{$.Site.Domain}}">
              <input type="hidden" name="user" value="{{.Login}}">
              <input type="hidden" name="target" value="{{$.Site.Domain}}">
              <button class="btn mini danger" name="action" value="user-delete">{{icon "trash"}}</button>
            </form>
          </div></td>
        </tr>{{else}}<tr><td colspan="3" class="muted">No users yet — add one below.</td></tr>{{end}}
      </tbody>
    </table>
    <hr class="sep">
    <h4 class="muted" style="margin:0 0 8px">Add user to <span class="mono">{{.Name}}</span></h4>
    <form method="post" action="/databases">
      <input type="hidden" name="site" value="{{$.Site.Domain}}">
      <input type="hidden" name="target" value="{{.Name}}">
      <div class="row">
        <div class="field"><label>Username (letters, digits, _; max 32)</label><input class="input mono" name="user" placeholder="app_user" required></div>
        <div class="field"><label>Password (8+ chars, leave blank to auto-generate)</label><input class="input mono" name="password" type="text" placeholder="leave empty for random"></div>
        <button class="btn primary" name="action" value="user-create" style="align-self:flex-end">{{icon "plus"}} Add user</button>
      </div>
    </form>
  </div>
  {{else}}
    <div class="card muted">No databases for this site yet. Create one above — the suffix becomes part of the name (e.g. <span class="mono">{{.DBPrefix}}_main</span>).</div>
  {{end}}
{{end}}

{{if eq .Tab "wordpress"}}
  {{if .WPManage}}
    <div class="card">
      <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:10px">
        <div><h2 style="margin:0">{{icon "wordpress"}} WordPress detected</h2><p class="muted mono" style="margin:4px 0 0">{{.WPManage.Path}} · WP {{if .WPManage.Version}}{{.WPManage.Version}}{{else}}unknown{{end}} · {{.WPManage.SiteURL}}</p></div>
        <a class="btn primary" href="/wordpress?manage={{.Site.Domain}}">{{icon "settings"}} Full WordPress manager →</a>
      </div>
    </div>
    <div class="grid-2">
      <div class="card">
        <h3>{{icon "refresh"}} Quick: update core</h3>
        <form method="post" action="/wordpress"><input type="hidden" name="domain" value="{{.Site.Domain}}">
          <button class="btn primary" name="action" value="update-core">Update WordPress</button>
        </form>
      </div>
      <div class="card">
        <h3>{{icon "activity"}} Quick: flush cache</h3>
        <form method="post" action="/wordpress"><input type="hidden" name="domain" value="{{.Site.Domain}}">
          <button class="btn" name="action" value="flush-cache">Flush cache + rewrites</button>
        </form>
      </div>
    </div>
    <div class="card">
      <h3>{{icon "users"}} Users</h3>
      <table><thead><tr><th>ID</th><th>Login</th><th>Email</th><th>Roles</th></tr></thead><tbody>
        {{range .WPUsers}}<tr><td class="mono">{{.ID}}</td><td><strong>{{.Login}}</strong></td><td class="muted">{{.Email}}</td><td><span class="badge">{{.Roles}}</span></td></tr>
        {{else}}<tr><td colspan="4" class="muted">No users returned.</td></tr>{{end}}
      </tbody></table>
      <p class="muted" style="margin-top:10px">For per-user password resets, URL changes, and delete — open the <a href="/wordpress?manage={{.Site.Domain}}" style="color:#2563eb;font-weight:700">full manager</a>.</p>
    </div>
  {{else}}
    <div class="card">
      <h3>Install WordPress on {{.Site.Domain}}</h3>
      <form method="post" action="/wordpress">
        <input type="hidden" name="action" value="install">
        <div class="row">
          <div class="field"><label>Domain</label><input class="input" name="domain" value="{{.Site.Domain}}" required readonly></div>
          <div class="field"><label>Site title</label><input class="input" name="title" required></div>
        </div>
        <div class="row">
          <div class="field"><label>Admin username</label><input class="input" name="admin_user" required></div>
          <div class="field"><label>Admin password</label><input class="input" name="admin_pass" type="text" required></div>
          <div class="field"><label>Admin email</label><input class="input" name="admin_email" type="email" required></div>
        </div>
        <button class="btn primary">{{icon "plus"}} Install WordPress</button>
        <p class="muted" style="margin-top:8px">Creates the DB, downloads WP core, runs <span class="mono">wp config create</span> + <span class="mono">wp core install</span>, fixes file ownership.</p>
      </form>
    </div>
  {{end}}
{{end}}

{{if eq .Tab "ssl"}}
  <div class="card">
    <h3>{{icon "shield"}} Issue / renew certificate</h3>
    <form method="post" action="/ssl">
      <div class="row">
        <div class="field"><label>Domain</label><input class="input" name="domain" value="{{.Site.Domain}}" required></div>
        <div class="field"><label>Admin email</label><input class="input" name="email" type="email" placeholder="admin@{{.Site.Domain}}"></div>
      </div>
      <div class="actions">
        <button class="btn primary" name="action" value="issue">{{icon "shield"}} Install SSL</button>
        <button class="btn" name="action" value="renew">{{icon "refresh"}} Renew</button>
        <button class="btn danger" name="action" value="delete" data-confirm="Delete certificate?">{{icon "trash"}} Delete cert</button>
      </div>
      <p class="muted" style="margin-top:8px">If this domain sits behind Cloudflare proxy, switch the DNS record to <strong>grey-cloud (DNS only)</strong> before issuing, then flip back to orange once the cert is installed.</p>
    </form>
  </div>
  <div class="card" style="padding:0">
    <table>
      <thead><tr><th>Certificate</th><th>Expiry</th><th>Status</th><th>Days left</th></tr></thead>
      <tbody>
        {{range .Certificates}}<tr>
          <td class="mono"><strong>{{.Domain}}</strong></td>
          <td>{{.Expiry}}</td>
          <td>{{if eq .Status "valid"}}<span class="badge ok">{{icon "check"}} valid</span>{{else if eq .Status "expiring"}}<span class="badge warn">{{icon "alert"}} expiring</span>{{else}}<span class="badge bad">{{icon "alert"}} critical</span>{{end}}</td>
          <td>{{.Days}}d</td>
        </tr>{{else}}<tr><td class="muted" colspan="4">No certificates issued yet.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
{{end}}

{{if eq .Tab "php"}}
  <div class="grid">
    {{range .PHP}}<div class="stat">
      <div class="label">{{icon "cpu"}} PHP {{.Version}}</div>
      <div class="val">{{if eq .Status "active"}}<span class="ok">active</span>{{else}}<span class="bad">{{.Status}}</span>{{end}}</div>
      <div class="sub mono">{{.Service}}</div>
    </div>{{end}}
  </div>
  <div class="card">
    <h3>Switch PHP for {{.Site.Domain}}</h3>
    <form method="post" action="/php">
      <input type="hidden" name="domain" value="{{.Site.Domain}}">
      <div class="row">
        <select class="input" name="version">
          <option {{if eq .Site.PHPVersion "8.4"}}selected{{end}}>8.4</option>
          <option {{if eq .Site.PHPVersion "8.3"}}selected{{end}}>8.3</option>
          <option {{if eq .Site.PHPVersion "8.2"}}selected{{end}}>8.2</option>
          <option {{if eq .Site.PHPVersion "8.5"}}selected{{end}}>8.5</option>
        </select>
        <button class="btn primary">{{icon "refresh"}} Apply (rewrites vhost, preserves SSL)</button>
      </div>
      <p class="muted" style="margin-top:8px">Currently using <strong>PHP {{.Site.PHPVersion}}</strong>. The vhost is rewritten and nginx reloaded.</p>
    </form>
  </div>
{{end}}

{{if eq .Tab "backups"}}
  <div class="grid-2">
    <div class="card">
      <h2>{{icon "archive"}} Manual backup</h2>
      <p class="muted">Creates a zip with site files and database.sql when a site database exists.</p>
      <form method="post" action="/backups"><input type="hidden" name="domain" value="{{.Site.Domain}}">
        <button class="btn primary" name="action" value="create">{{icon "plus"}} Create backup now</button>
      </form>
    </div>
    <div class="card">
      <h2>{{icon "clock"}} Automatic backup</h2>
      <form method="post" action="/backups"><input type="hidden" name="domain" value="{{.Site.Domain}}">
        <div class="row">
          <div class="field"><label>Frequency</label><select class="input" name="frequency"><option value="daily">daily</option><option value="weekly">weekly</option><option value="monthly">monthly</option></select></div>
          <div class="field"><label>Hour</label><input class="input" name="hour" value="{{.Policy.Hour}}"></div>
        </div>
        <div class="row">
          <div class="field"><label>Keep</label><input class="input" name="keep" value="{{.Policy.Keep}}"></div>
          <div class="field"><label>Max load</label><input class="input" name="max_load" value="{{.Policy.MaxLoad}}"></div>
        </div>
        <button class="btn primary" name="action" value="policy">{{icon "check"}} Save policy</button>
      </form>
    </div>
  </div>
  <div class="card" style="padding:0">
    <table>
      <thead><tr><th>Backup</th><th>Created</th><th>Size</th><th class="right-col">Actions</th></tr></thead>
      <tbody>
        {{range .Backups}}<tr>
          <td class="mono">{{.Name}}</td><td>{{.Created}}</td><td>{{.Size}}</td>
          <td class="right-col"><div class="actions" style="justify-content:flex-end">
            <a class="btn mini primary" href="/backups?site={{.Domain}}&download={{.Name}}">{{icon "download"}}</a>
            <form method="post" action="/backups" data-confirm="Restore full site?"><input type="hidden" name="domain" value="{{.Domain}}"><input type="hidden" name="name" value="{{.Name}}"><input type="hidden" name="mode" value="full"><button class="btn mini" name="action" value="restore">Full</button></form>
            <form method="post" action="/backups" data-confirm="Permanently delete?"><input type="hidden" name="domain" value="{{.Domain}}"><input type="hidden" name="name" value="{{.Name}}"><button class="btn mini danger" name="action" value="delete">{{icon "trash"}}</button></form>
          </div></td>
        </tr>{{else}}<tr><td colspan="4" class="muted">No backups yet.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
{{end}}
{{end}}

{{define "backups"}}
<div class="page-head">
  <div><h1>Backups</h1><p class="muted mono">{{.Site.Domain}}</p></div>
  <a class="btn" href="/site?domain={{.Site.Domain}}">{{icon "chevronUp"}} Back</a>
</div>
{{if .Output}}<div class="alert ok">{{icon "info"}} {{.Output}}</div>{{end}}
<div class="grid-2">
  <div class="card">
    <h2>{{icon "archive"}} Manual backup</h2>
    <p class="muted">Creates a zip with site files and database.sql when a site database exists.</p>
    <form method="post"><input type="hidden" name="domain" value="{{.Site.Domain}}">
      <button class="btn primary" name="action" value="create">{{icon "plus"}} Create backup now</button>
    </form>
  </div>
  <div class="card">
    <h2>{{icon "clock"}} Automatic backup</h2>
    <form method="post"><input type="hidden" name="domain" value="{{.Site.Domain}}">
      <div class="row">
        <div class="field"><label>Frequency</label>
          <select class="input" name="frequency"><option value="daily">daily</option><option value="weekly">weekly</option><option value="monthly">monthly</option></select>
        </div>
        <div class="field"><label>Hour (0-23)</label><input class="input" name="hour" value="{{.Policy.Hour}}"></div>
      </div>
      <div class="row">
        <div class="field"><label>Keep copies</label><input class="input" name="keep" value="{{.Policy.Keep}}"></div>
        <div class="field"><label>Max load</label><input class="input" name="max_load" value="{{.Policy.MaxLoad}}"></div>
      </div>
      <button class="btn primary" name="action" value="policy">{{icon "check"}} Save policy</button>
    </form>
  </div>
</div>
<div class="card" style="padding:0">
  <table>
    <thead><tr><th>Backup</th><th>Created</th><th>Size</th><th class="right-col">Actions</th></tr></thead>
    <tbody>
      {{range .Backups}}<tr>
        <td class="mono">{{.Name}}</td>
        <td>{{.Created}}</td>
        <td>{{.Size}}</td>
        <td class="right-col"><div class="actions" style="justify-content:flex-end">
          <a class="btn mini primary" href="/backups?site={{.Domain}}&download={{.Name}}">{{icon "download"}} Download</a>
          <form method="post" data-confirm="Restore full site from this backup?"><input type="hidden" name="domain" value="{{.Domain}}"><input type="hidden" name="name" value="{{.Name}}"><input type="hidden" name="mode" value="full"><button class="btn mini" name="action" value="restore">Restore full</button></form>
          <form method="post" data-confirm="Restore only the files?"><input type="hidden" name="domain" value="{{.Domain}}"><input type="hidden" name="name" value="{{.Name}}"><input type="hidden" name="mode" value="files"><button class="btn mini" name="action" value="restore">Files</button></form>
          <form method="post" data-confirm="Restore only the database?"><input type="hidden" name="domain" value="{{.Domain}}"><input type="hidden" name="name" value="{{.Name}}"><input type="hidden" name="mode" value="database"><button class="btn mini" name="action" value="restore">DB</button></form>
          <form method="post" data-confirm="Permanently delete this backup?"><input type="hidden" name="domain" value="{{.Domain}}"><input type="hidden" name="name" value="{{.Name}}"><button class="btn mini danger" name="action" value="delete">{{icon "trash"}}</button></form>
        </div></td>
      </tr>{{else}}<tr><td colspan="4" class="muted">No backups yet.</td></tr>{{end}}
    </tbody>
  </table>
</div>
{{end}}

{{define "wordpress"}}
<div class="page-head">
  <div><h1>{{icon "wordpress"}} WordPress</h1><p>Install and manage WordPress sites via WP-CLI.</p></div>
  <span class="badge info">WP-CLI</span>
</div>
{{if .Output}}<pre class="mono">{{.Output}}</pre>{{end}}

{{if .Manage}}
<div class="card">
  <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:10px;margin-bottom:8px">
    <div>
      <h2 style="margin:0">{{icon "wordpress"}} Manage {{.Manage.Domain}}</h2>
      <p class="muted mono" style="margin:4px 0 0">{{.Manage.Path}} · WP {{if .Manage.Version}}{{.Manage.Version}}{{else}}unknown{{end}} · {{.Manage.SiteURL}}</p>
    </div>
    <a class="btn" href="/wordpress">{{icon "chevronUp"}} Back to list</a>
  </div>
</div>
<div class="grid-2">
  <div class="card">
    <h3>{{icon "refresh"}} Update core</h3>
    <p class="muted">Run <span class="mono">wp core update</span> + <span class="mono">update-db</span>.</p>
    <form method="post"><input type="hidden" name="domain" value="{{.Manage.Domain}}">
      <button class="btn primary" name="action" value="update-core">{{icon "refresh"}} Update WordPress</button>
    </form>
  </div>
  <div class="card">
    <h3>{{icon "activity"}} Flush cache & rewrite rules</h3>
    <p class="muted">Run <span class="mono">wp cache flush</span> + <span class="mono">rewrite flush</span>.</p>
    <form method="post"><input type="hidden" name="domain" value="{{.Manage.Domain}}">
      <button class="btn" name="action" value="flush-cache">{{icon "refresh"}} Flush cache</button>
    </form>
  </div>
  <div class="card">
    <h3>{{icon "key"}} Reset user password</h3>
    <form method="post"><input type="hidden" name="domain" value="{{.Manage.Domain}}">
      <div class="field"><label>WordPress username</label><input class="input" name="user" placeholder="admin" required></div>
      <div class="field"><label>New password (8+ characters)</label><input class="input" name="password" type="text" minlength="8" required></div>
      <button class="btn primary" name="action" value="reset-pass">{{icon "key"}} Reset password</button>
    </form>
  </div>
  <div class="card">
    <h3>{{icon "globe"}} Change site URL</h3>
    <p class="muted">Updates <span class="mono">siteurl</span> + <span class="mono">home</span> and search-replaces the old URL across the DB.</p>
    <form method="post"><input type="hidden" name="domain" value="{{.Manage.Domain}}">
      <div class="field"><label>New URL</label><input class="input" name="url" placeholder="https://{{.Manage.Domain}}" value="{{.Manage.SiteURL}}" required></div>
      <button class="btn" name="action" value="change-url">{{icon "check"}} Apply new URL</button>
    </form>
  </div>
</div>
<div class="card">
  <h3>{{icon "users"}} Users</h3>
  <table>
    <thead><tr><th>ID</th><th>Login</th><th>Email</th><th>Roles</th></tr></thead>
    <tbody>
      {{range .Users}}<tr><td class="mono">{{.ID}}</td><td><strong>{{.Login}}</strong></td><td class="muted">{{.Email}}</td><td><span class="badge">{{.Roles}}</span></td></tr>
      {{else}}<tr><td colspan="4" class="muted">No users returned (WP-CLI may have failed for this install).</td></tr>{{end}}
    </tbody>
  </table>
</div>
<div class="card" style="border-color:#fecaca;background:#fff7f7">
  <h3 class="bad">{{icon "alert"}} Danger zone</h3>
  <p class="muted">Removes all files under <span class="mono">{{.Manage.Path}}</span> and drops the matching MariaDB database. The Nginx vhost and the parent site directory are kept so you can reuse the domain.</p>
  <form method="post" data-confirm="DELETE the WordPress install at {{.Manage.Path}} and DROP its database? This cannot be undone.">
    <input type="hidden" name="domain" value="{{.Manage.Domain}}">
    <button class="btn danger" name="action" value="delete">{{icon "trash"}} Delete WordPress install</button>
  </form>
</div>
{{else}}
<div class="card">
  <h3>Install new WordPress site</h3>
  <form method="post"><input type="hidden" name="action" value="install">
    <div class="row">
      <div class="field"><label>Domain</label><input class="input" name="domain" placeholder="example.com" value="{{.Site}}" required></div>
      <div class="field"><label>Site title</label><input class="input" name="title" required></div>
    </div>
    <div class="row">
      <div class="field"><label>Admin username</label><input class="input" name="admin_user" required></div>
      <div class="field"><label>Admin password</label><input class="input" name="admin_pass" type="text" required></div>
      <div class="field"><label>Admin email</label><input class="input" name="admin_email" type="email" required></div>
    </div>
    <button class="btn primary">{{icon "plus"}} Install WordPress</button>
  </form>
</div>
<div class="card" style="padding:0">
  <table>
    <thead><tr><th>Domain</th><th>Path</th><th>WP version</th><th>Site URL</th><th>Status</th><th class="right-col">Manage</th></tr></thead>
    <tbody>
      {{range .Installs}}<tr>
        <td><strong>{{.Domain}}</strong></td>
        <td class="mono muted">{{.Path}}</td>
        <td>{{if .Version}}<span class="badge info">{{.Version}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
        <td class="mono muted">{{.SiteURL}}</td>
        <td><span class="badge ok">{{icon "check"}} {{.Status}}</span></td>
        <td class="right-col"><a class="btn mini primary" href="/wordpress?manage={{.Domain}}">{{icon "settings"}} Manage</a></td>
      </tr>
      {{else}}<tr><td colspan="6" class="muted">No WordPress installs detected under /var/www. Install one above to get started.</td></tr>{{end}}
    </tbody>
  </table>
</div>
{{end}}
{{end}}

{{define "files"}}
<div class="page-head">
  <div><h1>{{icon "folderOpen"}} File Manager</h1><p class="muted">Right-click a row for actions — or use the toolbar.</p></div>
  <span class="badge info mono">{{.Path}}</span>
</div>
{{if .Output}}<div class="alert info">{{icon "info"}} {{.Output}}</div>{{end}}

<div class="crumbs">
  {{range $i,$c := .Crumbs}}{{if gt $i 0}}<span class="sep">/</span>{{end}}<a href="/files?path={{$c.Path}}">{{if eq $c.Name "/"}}{{icon "home"}}{{else}}{{$c.Name}}{{end}}</a>{{end}}
</div>

<div class="fm-toolbar">
  <button class="btn" onclick="openModal('m-mkdir')">{{icon "folder"}} New folder</button>
  <button class="btn" onclick="openModal('m-touch')">{{icon "file"}} New file</button>
  <button class="btn" onclick="openModal('m-upload')">{{icon "upload"}} Upload</button>
  <a class="btn" href="/files?path={{.Path}}">{{icon "refresh"}} Refresh</a>
  <div class="grow"></div>
  <span class="badge">{{len .Items}} item(s)</span>
</div>

<div class="card" style="padding:0">
  <table class="fm-table">
    <thead><tr><th>Name</th><th>Size</th><th>Mode</th><th>Modified</th><th class="right-col">Quick actions</th></tr></thead>
    <tbody id="fm-rows">
      {{range .Items}}<tr data-name="{{.Name}}" data-path="{{.Path}}" data-kind="{{.Kind}}" data-mode="{{.Mode}}">
        <td>
          {{if eq .Kind "dir"}}<a href="/files?path={{.Path}}" class="file-name dir"><span class="ic">{{icon "folder"}}</span>{{.Name}}</a>
          {{else}}<span class="file-name"><span class="ic">{{icon "file"}}</span>{{.Name}}</span>{{end}}
        </td>
        <td class="muted mono">{{.Size}}</td>
        <td class="muted mono">{{.Mode}}</td>
        <td class="muted">{{.ModTime}}</td>
        <td class="right-col">
          {{if eq .Kind "file"}}<a class="btn mini" href="/files?path={{$.Path}}&download={{.Path}}">{{icon "download"}}</a>{{end}}
          <button class="btn mini" onclick="fmAction('rename', this)">{{icon "edit"}}</button>
          <button class="btn mini" onclick="fmAction('chmod', this)">{{icon "shield"}}</button>
          <button class="btn mini danger" onclick="fmAction('delete', this)">{{icon "trash"}}</button>
        </td>
      </tr>{{else}}<tr><td colspan="5" class="muted">Folder is empty.</td></tr>{{end}}
    </tbody>
  </table>
</div>

<!-- Context Menu -->
<div class="ctxmenu" id="ctxmenu">
  <button onclick="fmAction('open')" data-files-hide>{{icon "folderOpen"}} Open</button>
  <button onclick="fmAction('download')" data-dirs-hide>{{icon "download"}} Download</button>
  <div class="sep"></div>
  <button onclick="fmAction('rename')">{{icon "edit"}} Rename</button>
  <button onclick="fmAction('chmod')">{{icon "shield"}} Permissions (chmod)</button>
  <button onclick="fmAction('copy')">{{icon "copy"}} Copy to…</button>
  <button onclick="fmAction('move')">{{icon "move"}} Move to…</button>
  <div class="sep"></div>
  <button class="danger" onclick="fmAction('delete')">{{icon "trash"}} Delete</button>
</div>

<!-- Modals -->
<div class="modal-bg" id="m-mkdir"><div class="modal">
  <h3>New folder</h3><p class="muted">Create a folder in <span class="mono">{{.Path}}</span>.</p>
  <form method="post"><input type="hidden" name="path" value="{{.Path}}"><input type="hidden" name="action" value="mkdir">
    <div class="field"><label>Folder name</label><input class="input" name="name" required autofocus></div>
    <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-mkdir')">Cancel</button><button class="btn primary">{{icon "plus"}} Create</button></div>
  </form>
</div></div>

<div class="modal-bg" id="m-touch"><div class="modal">
  <h3>New file</h3><p class="muted">Create an empty file in <span class="mono">{{.Path}}</span>.</p>
  <form method="post"><input type="hidden" name="path" value="{{.Path}}"><input type="hidden" name="action" value="touch">
    <div class="field"><label>File name</label><input class="input" name="name" required autofocus></div>
    <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-touch')">Cancel</button><button class="btn primary">{{icon "plus"}} Create</button></div>
  </form>
</div></div>

<div class="modal-bg" id="m-upload"><div class="modal">
  <h3>Upload files</h3><p class="muted">Upload one or more files to <span class="mono">{{.Path}}</span> (max 64 MB).</p>
  <form method="post" enctype="multipart/form-data"><input type="hidden" name="path" value="{{.Path}}"><input type="hidden" name="action" value="upload">
    <div class="field"><label>Choose files</label><input class="input" type="file" name="upload" multiple required></div>
    <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-upload')">Cancel</button><button class="btn primary">{{icon "upload"}} Upload</button></div>
  </form>
</div></div>

<div class="modal-bg" id="m-rename"><div class="modal">
  <h3>Rename</h3><p class="muted">Rename <span class="mono" id="rn-from"></span>.</p>
  <form method="post"><input type="hidden" name="path" value="{{.Path}}"><input type="hidden" name="action" value="rename"><input type="hidden" name="target" id="rn-target">
    <div class="field"><label>New name</label><input class="input" name="dest" id="rn-dest" required></div>
    <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-rename')">Cancel</button><button class="btn primary">{{icon "check"}} Rename</button></div>
  </form>
</div></div>

<div class="modal-bg" id="m-chmod"><div class="modal">
  <h3>Change permissions</h3><p class="muted"><span class="mono" id="cm-from"></span> — current mode <span class="mono" id="cm-cur"></span></p>
  <form method="post"><input type="hidden" name="path" value="{{.Path}}"><input type="hidden" name="action" value="chmod"><input type="hidden" name="target" id="cm-target">
    <div class="field"><label>Octal mode</label><input class="input mono" name="mode" id="cm-mode" placeholder="755" pattern="[0-7]{3,4}" required></div>
    <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-chmod')">Cancel</button><button class="btn primary">{{icon "check"}} Apply</button></div>
  </form>
</div></div>

<div class="modal-bg" id="m-copy"><div class="modal">
  <h3>Copy</h3><p class="muted">Copy <span class="mono" id="cp-from"></span> to a new path under <span class="mono">/var/www</span>.</p>
  <form method="post"><input type="hidden" name="path" value="{{.Path}}"><input type="hidden" name="action" value="copy"><input type="hidden" name="target" id="cp-target">
    <div class="field"><label>Destination path</label><input class="input mono" name="dest" id="cp-dest" placeholder="/site.com/htdocs/new-name" required></div>
    <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-copy')">Cancel</button><button class="btn primary">{{icon "copy"}} Copy</button></div>
  </form>
</div></div>

<div class="modal-bg" id="m-move"><div class="modal">
  <h3>Move</h3><p class="muted">Move <span class="mono" id="mv-from"></span> to a new path under <span class="mono">/var/www</span>.</p>
  <form method="post"><input type="hidden" name="path" value="{{.Path}}"><input type="hidden" name="action" value="move"><input type="hidden" name="target" id="mv-target">
    <div class="field"><label>Destination path</label><input class="input mono" name="dest" id="mv-dest" placeholder="/site.com/htdocs/new-location" required></div>
    <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-move')">Cancel</button><button class="btn primary">{{icon "move"}} Move</button></div>
  </form>
</div></div>

<form method="post" id="fm-delete-form" style="display:none"><input type="hidden" name="path" value="{{.Path}}"><input type="hidden" name="action" value="delete"><input type="hidden" name="target" id="del-target"></form>

<script>
(function(){
  var ctx=document.getElementById('ctxmenu');
  var currentRow=null;
  function rowFrom(el){ while(el && el.tagName!=='TR'){ el=el.parentElement; } return el; }
  document.getElementById('fm-rows').addEventListener('contextmenu',function(e){
    var tr=rowFrom(e.target);
    if(!tr || !tr.dataset.path) return;
    e.preventDefault();
    currentRow=tr;
    var kind=tr.dataset.kind;
    ctx.querySelectorAll('[data-files-hide]').forEach(function(b){ b.style.display=(kind==='dir')?'flex':'none'; });
    ctx.querySelectorAll('[data-dirs-hide]').forEach(function(b){ b.style.display=(kind==='file')?'flex':'none'; });
    ctx.style.left=Math.min(e.clientX, window.innerWidth-230)+'px';
    ctx.style.top=Math.min(e.clientY, window.innerHeight-340)+'px';
    ctx.classList.add('show');
  });
  document.addEventListener('click',function(e){
    if(!ctx.contains(e.target)) ctx.classList.remove('show');
  });
  document.addEventListener('keydown',function(e){ if(e.key==='Escape') ctx.classList.remove('show'); });

  window.fmAction=function(action, btn){
    var tr=currentRow;
    if(btn){ tr=rowFrom(btn); }
    ctx.classList.remove('show');
    if(!tr) return;
    var name=tr.dataset.name, path=tr.dataset.path, kind=tr.dataset.kind, mode=tr.dataset.mode;
    var base="{{.Path}}";
    var parent=base.replace(/\/+$/,'');
    if(action==='open'){ if(kind==='dir') window.location='/files?path='+encodeURIComponent(path); return; }
    if(action==='download'){ if(kind==='file') window.location='/files?path='+encodeURIComponent(base)+'&download='+encodeURIComponent(path); return; }
    if(action==='delete'){
      if(!confirm('Permanently delete '+name+'?')) return;
      document.getElementById('del-target').value=path;
      document.getElementById('fm-delete-form').submit(); return;
    }
    if(action==='rename'){
      document.getElementById('rn-from').textContent=name;
      document.getElementById('rn-target').value=path;
      document.getElementById('rn-dest').value=name;
      openModal('m-rename'); return;
    }
    if(action==='chmod'){
      document.getElementById('cm-from').textContent=name;
      document.getElementById('cm-cur').textContent=mode;
      document.getElementById('cm-target').value=path;
      document.getElementById('cm-mode').value=mode;
      openModal('m-chmod'); return;
    }
    if(action==='copy'){
      document.getElementById('cp-from').textContent=path;
      document.getElementById('cp-target').value=path;
      document.getElementById('cp-dest').value=(parent==='/'?'':parent)+'/'+name+'.copy';
      openModal('m-copy'); return;
    }
    if(action==='move'){
      document.getElementById('mv-from').textContent=path;
      document.getElementById('mv-target').value=path;
      document.getElementById('mv-dest').value=path;
      openModal('m-move'); return;
    }
  };
})();
</script>
{{end}}

{{define "databases"}}
<div class="page-head">
  <div><h1>{{icon "database"}} Databases</h1><p>MariaDB / MySQL databases managed by hostQ.</p></div>
  <span class="badge info">MariaDB</span>
</div>
{{if .Created}}<div class="alert ok">
  {{icon "check"}}
  <div>
    <strong>Database created:</strong> <span class="mono">{{.Created}}</span><br>
    <strong>User:</strong> <span class="mono">{{.User}}</span> · <strong>Password:</strong> <span class="mono">{{.Password}}</span>
    <div class="muted" style="font-weight:500">Save this password now — it is shown only once.</div>
  </div>
</div>{{end}}
<div class="card">
  <h3>Create database</h3>
  <form method="post"><div class="row">
    <input class="input" name="name" placeholder="project_name" value="{{.Site}}" required>
    <button class="btn primary" name="action" value="create">{{icon "plus"}} Create database</button>
  </div>
  <p class="muted" style="margin-top:8px">A database and matching user are created with a generated password. Database names are auto-prefixed with <span class="mono">hostq_</span>.</p>
  </form>
</div>
<div class="card" style="padding:0">
  <table>
    <thead><tr><th>Database</th><th class="right-col">Actions</th></tr></thead>
    <tbody>
      {{range .Databases}}<tr>
        <td class="mono">{{.Name}}</td>
        <td class="right-col"><form method="post" data-confirm="Permanently drop this database and its user?">
          <input type="hidden" name="target" value="{{.Name}}">
          <button class="btn mini danger" name="action" value="delete">{{icon "trash"}} Drop</button>
        </form></td>
      </tr>{{else}}<tr><td class="muted" colspan="2">No user databases found, or mysql CLI is not available.</td></tr>{{end}}
    </tbody>
  </table>
</div>
{{end}}

{{define "php"}}
<div class="page-head">
  <div><h1>{{icon "cpu"}} PHP Manager</h1><p>Switch PHP-FPM versions per site.</p></div>
  <span class="badge info">FPM</span>
</div>
<div class="grid">
  {{range .PHP}}<div class="stat">
    <div class="label">{{icon "cpu"}} PHP {{.Version}}</div>
    <div class="val">{{if eq .Status "active"}}<span class="ok">active</span>{{else}}<span class="bad">{{.Status}}</span>{{end}}</div>
    <div class="sub mono">{{.Service}}</div>
  </div>{{end}}
</div>
<div class="card">
  <h3>Switch PHP for a site</h3>
  <form method="post"><div class="row">
    <select class="input" name="domain">{{range .Sites}}<option value="{{.Domain}}">{{.Domain}} — currently {{.PHPVersion}}</option>{{end}}</select>
    <select class="input" name="version"><option>8.4</option><option>8.3</option><option>8.2</option><option>8.5</option></select>
    <button class="btn primary">{{icon "refresh"}} Apply</button>
  </div></form>
</div>
{{end}}

{{define "ssl"}}
<div class="page-head">
  <div><h1>{{icon "shield"}} SSL Certificates</h1><p>Powered by Let's Encrypt + Certbot.</p></div>
  <span class="badge info">Let's Encrypt</span>
</div>
{{if .Output}}<pre class="mono">{{.Output}}</pre>{{end}}
<div class="card">
  <h3>Issue / renew certificate</h3>
  <form method="post">
    <div class="row">
      <div class="field"><label>Domain</label><input class="input" name="domain" placeholder="example.com" value="{{.Site}}" required></div>
      <div class="field"><label>Admin email</label><input class="input" name="email" placeholder="admin@example.com" type="email"></div>
    </div>
    <div class="actions">
      <button class="btn primary" name="action" value="issue">{{icon "shield"}} Install SSL</button>
      <button class="btn" name="action" value="renew">{{icon "refresh"}} Renew</button>
      <button class="btn danger" name="action" value="delete" data-confirm="Permanently delete this certificate?">{{icon "trash"}} Delete</button>
    </div>
    <p class="muted" style="margin-top:8px">If Nginx still references a missing certificate, hostQ repairs the vhost automatically before running certbot.</p>
  </form>
</div>
<div class="card" style="padding:0">
  <table>
    <thead><tr><th>Certificate</th><th>Expiry</th><th>Status</th><th>Days left</th></tr></thead>
    <tbody>
      {{range .Certificates}}<tr>
        <td class="mono"><strong>{{.Domain}}</strong></td>
        <td>{{.Expiry}}</td>
        <td>{{if eq .Status "valid"}}<span class="badge ok">{{icon "check"}} valid</span>{{else if eq .Status "expiring"}}<span class="badge warn">{{icon "alert"}} expiring</span>{{else}}<span class="badge bad">{{icon "alert"}} critical</span>{{end}}</td>
        <td>{{.Days}}d</td>
      </tr>{{else}}<tr><td class="muted" colspan="4">No certificates found.</td></tr>{{end}}
    </tbody>
  </table>
</div>
{{end}}

{{define "services"}}
<div class="page-head">
  <div><h1>{{icon "server"}} Services</h1><p>Start, stop and restart server daemons.</p></div>
</div>
<div class="card" style="padding:0">
  <table>
    <thead><tr><th>Service</th><th>Systemd unit</th><th>Status</th><th class="right-col">Actions</th></tr></thead>
    <tbody>
      {{range .Services}}<tr>
        <td><strong>{{.Name}}</strong></td>
        <td class="mono muted">{{.Systemd}}</td>
        <td>{{if eq .Status "active"}}<span class="badge ok">{{icon "check"}} active</span>{{else}}<span class="badge bad">{{icon "alert"}} {{.Status}}</span>{{end}}</td>
        <td class="right-col"><form method="post" style="display:inline-flex;gap:6px">
          <input type="hidden" name="id" value="{{.ID}}">
          <button class="btn mini" name="action" value="restart">{{icon "refresh"}} Restart</button>
          <button class="btn mini primary" name="action" value="start">{{icon "play"}} Start</button>
          <button class="btn mini danger" name="action" value="stop">{{icon "stop"}} Stop</button>
        </form></td>
      </tr>{{end}}
    </tbody>
  </table>
</div>
{{end}}

{{define "cron"}}
<div class="page-head">
  <div><h1>{{icon "clock"}} Cron Manager</h1><p>Manage scheduled commands created by hostQ.</p></div>
</div>
{{if .Output}}<div class="alert info">{{icon "info"}} {{.Output}}</div>{{end}}
<div class="card">
  <h3>Add cron job</h3>
  <form method="post">
    <div class="row">
      <div class="field"><label>Job name</label><input class="input" name="name" placeholder="daily clean cache"></div>
      <div class="field"><label>User</label><select class="input" name="user"><option>root</option><option>www-data</option></select></div>
    </div>
    <div class="row">
      <div class="field"><label>Minute</label><input class="input mono" name="minute" placeholder="*"></div>
      <div class="field"><label>Hour</label><input class="input mono" name="hour" placeholder="*"></div>
      <div class="field"><label>Day</label><input class="input mono" name="day" placeholder="*"></div>
      <div class="field"><label>Month</label><input class="input mono" name="month" placeholder="*"></div>
      <div class="field"><label>Weekday</label><input class="input mono" name="weekday" placeholder="*"></div>
    </div>
    <div class="field"><label>Command</label><input class="input mono" name="command" placeholder="/usr/bin/php /var/www/site/htdocs/cron.php"></div>
    <button class="btn primary" name="action" value="create">{{icon "plus"}} Add cron job</button>
    <p class="muted" style="margin-top:8px">Fields accept <span class="mono">*</span>, a number, or step syntax like <span class="mono">*/15</span>. Stored in <span class="mono">/etc/cron.d/hostq-user-jobs</span>.</p>
  </form>
</div>
<div class="card" style="padding:0">
  <table>
    <thead><tr><th>Name</th><th>Schedule</th><th>User</th><th>Command</th><th>Source</th><th class="right-col"></th></tr></thead>
    <tbody>
      {{range .Jobs}}<tr>
        <td>{{.Name}}</td>
        <td class="mono">{{.Schedule}}</td>
        <td>{{.User}}</td>
        <td class="mono muted">{{.Command}}</td>
        <td>{{if eq .Source "managed"}}<span class="badge info">managed</span>{{else}}<span class="badge">{{.Source}}</span>{{end}}</td>
        <td class="right-col">{{if eq .Source "managed"}}<form method="post" data-confirm="Delete this cron job?"><input type="hidden" name="id" value="{{.ID}}"><button class="btn mini danger" name="action" value="delete">{{icon "trash"}}</button></form>{{end}}</td>
      </tr>{{else}}<tr><td colspan="6" class="muted">No cron jobs found.</td></tr>{{end}}
    </tbody>
  </table>
</div>
{{end}}

{{define "account"}}
<div class="page-head">
  <div><h1>{{icon "users"}} Account</h1><p>Manage your panel administrator credentials.</p></div>
</div>
{{if .Output}}<div class="alert info">{{icon "info"}} {{.Output}}</div>{{end}}
<div class="grid-2">
  <div class="card">
    <h2>{{icon "info"}} Current account</h2>
    <p class="muted" style="margin-top:4px">Logged in as <strong>{{if .Account}}{{.Account.Username}}{{else}}admin{{end}}</strong> · role <span class="badge">{{if .Account}}{{.Account.Role}}{{else}}admin{{end}}</span></p>
  </div>
  <div class="card">
    <h2>{{icon "key"}} Change password</h2>
    <form method="post">
      <div class="field"><label>Current password</label><input class="input" type="password" name="current" required></div>
      <div class="field"><label>New password</label><input class="input" type="password" name="next" minlength="10" required></div>
      <div class="field"><label>Confirm new password</label><input class="input" type="password" name="confirm" minlength="10" required></div>
      <button class="btn primary">{{icon "check"}} Update password</button>
    </form>
  </div>
</div>
{{end}}

{{define "audit"}}
<div class="page-head">
  <div><h1>{{icon "activity"}} Audit Log</h1><p>The last 200 audit entries — newest first.</p></div>
</div>
<div class="card" style="padding:0">
  <table>
    <thead><tr><th>Timestamp</th><th>Action</th><th>Status</th><th>Target</th></tr></thead>
    <tbody>
      {{range .Entries}}<tr>
        <td class="mono">{{.Timestamp}}</td>
        <td><span class="badge">{{.Action}}</span></td>
        <td>{{if eq .Status "success"}}<span class="badge ok">{{icon "check"}} {{.Status}}</span>{{else}}<span class="badge bad">{{icon "alert"}} {{.Status}}</span>{{end}}</td>
        <td class="mono muted">{{.Target}}</td>
      </tr>{{else}}<tr><td colspan="4" class="muted">No audit entries yet.</td></tr>{{end}}
    </tbody>
  </table>
</div>
{{end}}
`
