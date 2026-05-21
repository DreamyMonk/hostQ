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
  --bg:#0a0f1c;--panel:#0d1424;--panel-2:#172033;--text:#e6eaf2;--muted:#7d8aa3;--line:#172033;
  --card:#ffffff;--card-line:#eef1f6;--card-line-2:#e6eaf0;--ink:#0b1220;--ink-muted:#5b6b85;
  --page:#f6f8fb;--surface:#fff;--surface-2:#fbfcfe;--surface-hover:#f4f6fb;
  --brand:#3b82f6;--brand-2:#2563eb;--accent:#06b6d4;
  --ok:#16a34a;--bad:#dc2626;--warn:#d97706;
  --radius:10px;--shadow:0 1px 1px rgba(15,23,42,.03),0 1px 3px rgba(15,23,42,.04);
  --shadow-lg:0 10px 30px rgba(15,23,42,.08);
}
[data-theme="dark"]{
  --card:#0f1729;--card-line:#1c2640;--card-line-2:#243153;--ink:#e6eaf2;--ink-muted:#8e9bb5;
  --page:#070b15;--surface:#0f1729;--surface-2:#121b31;--surface-hover:#172033;
  --shadow:0 1px 2px rgba(0,0,0,.35),0 0 0 1px rgba(255,255,255,.02);
  --shadow-lg:0 12px 40px rgba(0,0,0,.45);
}
*{box-sizing:border-box}
html,body{margin:0;padding:0;background:var(--page);color:var(--ink);font-family:Inter,ui-sans-serif,system-ui,Segoe UI,Roboto,sans-serif;font-size:13.5px;line-height:1.55;-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale}
a{color:inherit;text-decoration:none}
button{font-family:inherit;font-size:inherit}
svg{flex:none;vertical-align:middle}
.mono{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:12.5px}
.muted{color:var(--ink-muted)}
.ok{color:var(--ok)}.bad{color:var(--bad)}.warn{color:var(--warn)}

/* shell */
.shell{display:grid;grid-template-columns:232px 1fr;min-height:100vh}
aside.side{background:var(--bg);color:var(--text);padding:16px 10px;position:sticky;top:0;height:100vh;overflow-y:auto;border-right:1px solid #0a1020;display:flex;flex-direction:column}
.brand{display:flex;align-items:center;gap:10px;padding:6px 8px 16px;font-weight:800;font-size:16px;color:#fff;letter-spacing:-.01em}
.brand .mark{width:30px;height:30px;border-radius:8px;background:linear-gradient(135deg,#3b82f6,#06b6d4);display:grid;place-items:center;color:#fff;box-shadow:0 4px 12px rgba(59,130,246,.30)}
.brand .mark svg{color:#fff;width:16px;height:16px}
.brand small{display:block;font-size:10px;font-weight:600;color:var(--muted);letter-spacing:.10em;text-transform:uppercase;margin-top:1px}
.navgroup{margin:14px 0 4px;padding:0 10px;font-size:10px;font-weight:700;letter-spacing:.12em;text-transform:uppercase;color:#5b6477}
.nav a{display:flex;align-items:center;gap:10px;padding:7px 10px;border-radius:7px;color:#c5cde0;font-weight:600;font-size:13px;margin-bottom:1px;transition:background .12s,color .12s}
.nav a:hover{background:rgba(255,255,255,.04);color:#fff}
.nav a.active{background:rgba(59,130,246,.14);color:#fff}
.nav a svg{opacity:.85;width:16px;height:16px}
.side-foot{margin-top:auto;padding:12px 10px 4px;border-top:1px solid rgba(255,255,255,.05);font-size:11px;color:#7d8aa3}
.side-foot .row{display:flex;align-items:center;justify-content:space-between;gap:8px}

/* topbar */
main.main{min-width:0}
.topbar{position:sticky;top:0;z-index:10;background:var(--surface);border-bottom:1px solid var(--card-line);height:56px;display:flex;align-items:center;justify-content:space-between;padding:0 22px}
.topbar h1{margin:0;font-size:15px;font-weight:700;display:flex;align-items:center;gap:8px;letter-spacing:-.01em;color:var(--ink)}
.topbar h1 svg{opacity:.5;width:14px;height:14px}
.topbar .right{display:flex;align-items:center;gap:8px;color:var(--ink-muted);font-size:12.5px}
.topbar .right .chip{display:inline-flex;align-items:center;gap:6px;padding:4px 10px;border-radius:999px;background:var(--surface-hover);color:var(--ink-muted);font-weight:600;font-size:12px}
.topbar .right .chip svg{width:12px;height:12px}
.iconbtn{width:32px;height:32px;border-radius:8px;border:1px solid var(--card-line);background:var(--surface);color:var(--ink-muted);display:grid;place-items:center;cursor:pointer;transition:background .12s,color .12s,border-color .12s}
.iconbtn:hover{background:var(--surface-hover);color:var(--ink);border-color:var(--card-line-2)}
.iconbtn svg{width:14px;height:14px}
.kbd{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:10.5px;padding:1px 5px;border-radius:4px;border:1px solid var(--card-line);background:var(--surface-2);color:var(--ink-muted)}
.searchbtn{display:flex;align-items:center;gap:8px;padding:6px 10px;border-radius:8px;border:1px solid var(--card-line);background:var(--surface);color:var(--ink-muted);font-size:12.5px;cursor:pointer;transition:background .12s,border-color .12s}
.searchbtn:hover{background:var(--surface-hover);border-color:var(--card-line-2)}
.searchbtn .kbd{margin-left:4px}
.content{padding:22px}

/* cards */
.card{background:var(--card);border:1px solid var(--card-line);border-radius:12px;padding:18px;margin-bottom:12px;box-shadow:var(--shadow);color:var(--ink)}
.card h2{margin:0 0 4px;font-size:15px;font-weight:700;letter-spacing:-.01em;color:var(--ink)}
.card h3{margin:0 0 10px;font-size:12px;font-weight:700;color:var(--ink-muted);text-transform:uppercase;letter-spacing:.08em}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px}
.grid-2{display:grid;grid-template-columns:repeat(auto-fit,minmax(340px,1fr));gap:12px}

/* stat cards */
.stat{display:flex;flex-direction:column;gap:6px;background:var(--card);border:1px solid var(--card-line);border-radius:12px;padding:16px 18px;box-shadow:var(--shadow);transition:transform .12s,box-shadow .12s;color:var(--ink)}
.stat:hover{transform:translateY(-1px);box-shadow:var(--shadow-lg)}
.stat .label{display:flex;align-items:center;gap:8px;color:var(--ink-muted);font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:.08em}
.stat .label svg{opacity:.7;width:14px;height:14px}
.stat .val{font-size:24px;font-weight:800;color:var(--ink);letter-spacing:-.02em;line-height:1.1}
.stat .sub{font-size:12px;color:var(--ink-muted)}
.bar{height:5px;border-radius:99px;background:#eef2f7;overflow:hidden;margin-top:4px}
.bar > div{height:100%;background:linear-gradient(90deg,#3b82f6,#06b6d4);transition:width .5s ease}

/* buttons */
.btn{display:inline-flex;align-items:center;gap:6px;border:1px solid var(--card-line);background:var(--surface);color:var(--ink);border-radius:8px;padding:7px 12px;font-weight:600;font-size:13px;cursor:pointer;transition:background .12s,border-color .12s,transform .05s,box-shadow .12s}
.btn:hover{background:var(--surface-hover);border-color:var(--card-line-2)}
.btn:active{transform:translateY(1px)}
.btn svg{width:14px;height:14px}
.btn.primary{background:var(--brand-2);border-color:var(--brand-2);color:#fff;box-shadow:0 1px 2px rgba(37,99,235,.25)}
.btn.primary:hover{background:#1d4ed8;border-color:#1d4ed8;box-shadow:0 2px 6px rgba(37,99,235,.35)}
.btn.ghost{background:transparent;border-color:transparent}
.btn.ghost:hover{background:#f1f5f9}
.btn.danger{color:#b91c1c;border-color:#fde2e2;background:#fff7f7}
.btn.danger:hover{background:#fee2e2;border-color:#fbcfcf}
.btn.mini{padding:4px 9px;font-size:12px;border-radius:6px}
.btn.icon{padding:7px;border-radius:8px}
.actions{display:flex;gap:6px;flex-wrap:wrap}

/* inputs */
.input,select.input,textarea.input{width:100%;border:1px solid var(--card-line);background:var(--surface);border-radius:8px;padding:8px 11px;font-size:13.5px;color:var(--ink);outline:none;transition:border-color .12s,box-shadow .12s}
.input:hover{border-color:var(--card-line-2)}
.input:focus{border-color:#93c5fd;box-shadow:0 0 0 3px rgba(59,130,246,.12)}
.input::placeholder{color:var(--ink-muted);opacity:.7}
.field{display:flex;flex-direction:column;gap:6px;margin-bottom:10px}
.field label{font-size:11.5px;font-weight:700;color:var(--ink-muted);letter-spacing:.01em}
.row{display:flex;gap:10px;flex-wrap:wrap}
.row > *{flex:1;min-width:160px}

/* table */
table{width:100%;border-collapse:separate;border-spacing:0;background:var(--card);border:1px solid var(--card-line);border-radius:12px;overflow:hidden;color:var(--ink)}
th,td{padding:11px 14px;text-align:left;font-size:13px;border-bottom:1px solid var(--card-line);vertical-align:middle}
th{font-size:11px;font-weight:700;color:var(--ink-muted);text-transform:uppercase;letter-spacing:.08em;background:var(--surface-2)}
tbody tr:last-child td{border-bottom:none}
tbody tr:hover{background:var(--surface-2)}

/* badges */
.badge{display:inline-flex;align-items:center;gap:5px;border:1px solid var(--card-line);border-radius:999px;padding:2px 9px;font-size:11px;font-weight:700;background:var(--surface);color:var(--ink-muted);letter-spacing:.01em}
.badge svg{width:11px;height:11px}
.badge.ok{color:#166534;border-color:#bbf7d0;background:#f0fdf4}
.badge.bad{color:#991b1b;border-color:#fde2e2;background:#fef2f2}
.badge.warn{color:#92400e;border-color:#fde68a;background:#fffbeb}
.badge.info{color:#1e40af;border-color:#cfdcfb;background:#eff5ff}

/* page heading */
.page-head{display:flex;justify-content:space-between;align-items:flex-end;margin:0 0 18px;gap:10px;flex-wrap:wrap}
.page-head h1{margin:0;font-size:22px;font-weight:800;letter-spacing:-.02em;color:#0b1220}
.page-head p{margin:4px 0 0;color:var(--ink-muted);font-size:13px}

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
.ctxmenu{position:fixed;z-index:1000;min-width:210px;background:var(--card);color:var(--ink);border:1px solid var(--card-line);border-radius:10px;box-shadow:var(--shadow-lg);padding:6px;display:none}
.ctxmenu.show{display:block}
.ctxmenu button{width:100%;border:none;background:none;text-align:left;padding:8px 10px;border-radius:6px;display:flex;align-items:center;gap:10px;cursor:pointer;color:var(--ink);font-weight:600;font-size:13px}
.ctxmenu button:hover{background:var(--surface-hover)}
.ctxmenu .sep{height:1px;background:var(--card-line);margin:4px 2px}
.ctxmenu .danger{color:#b91c1c}

/* modal */
.modal-bg{position:fixed;inset:0;background:rgba(7,11,21,.55);backdrop-filter:blur(4px);z-index:900;display:none;align-items:center;justify-content:center;padding:20px}
.modal-bg.show{display:flex}
.modal{background:var(--card);color:var(--ink);border:1px solid var(--card-line);border-radius:12px;padding:20px;width:100%;max-width:480px;box-shadow:var(--shadow-lg)}
.modal h3{margin:0 0 4px;font-size:16px;font-weight:800;color:var(--ink)}
.modal p.muted{margin:0 0 14px}
.modal .modal-foot{display:flex;gap:8px;justify-content:flex-end;margin-top:6px}

/* command palette */
.modal.palette{max-width:560px;padding:0;overflow:hidden}
.palette-input{border:none;border-radius:0;border-bottom:1px solid var(--card-line);padding:14px 18px;font-size:15px;background:transparent}
.palette-input:focus{box-shadow:none;border-color:var(--card-line-2)}
.palette-list{max-height:50vh;overflow-y:auto;padding:6px}
.palette-cat{padding:8px 10px 4px;font-size:10.5px;font-weight:700;color:var(--ink-muted);text-transform:uppercase;letter-spacing:.10em}
.palette-row{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:9px 12px;border-radius:7px;color:var(--ink);font-weight:600;font-size:13.5px;cursor:pointer;text-decoration:none}
.palette-row:hover,.palette-row.on{background:var(--surface-hover)}
.palette-row.on{outline:1px solid var(--brand);outline-offset:-1px}
.palette-label{flex:1;min-width:0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.palette-hint{font-size:11px;color:var(--ink-muted);font-weight:600}
.palette-empty{padding:18px;text-align:center;color:var(--ink-muted);font-size:13px}
.palette-foot{display:flex;gap:14px;justify-content:flex-end;padding:8px 14px;border-top:1px solid var(--card-line);background:var(--surface-2);font-size:11px;color:var(--ink-muted)}

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

/* site-head — sticky-ish header for /site */
.site-head{display:flex;justify-content:space-between;align-items:center;gap:14px;flex-wrap:wrap;margin:0 0 16px}
.site-head-main{display:flex;align-items:center;gap:12px;min-width:0}
.site-head-main h1{margin:0;font-size:22px;font-weight:800;letter-spacing:-.02em}
.site-head-main p{margin:2px 0 0;font-size:12px}
.site-back{width:34px;height:34px;border-radius:8px;border:1px solid var(--card-line);background:#fff;display:grid;place-items:center;color:#475569;flex:none;transition:background .12s,color .12s}
.site-back:hover{background:#f4f6fb;color:#0b1220}
.site-back svg{transform:rotate(-90deg);width:14px;height:14px}
.site-head-meta{display:flex;align-items:center;gap:8px;flex-wrap:wrap}

/* per-card toolbar (between tab + cards) */
.toolbar{display:flex;align-items:center;justify-content:space-between;gap:10px;margin:0 0 12px;padding:0 2px;flex-wrap:wrap}
.toolbar .muted{font-size:12.5px}

/* credentials banner — replaces alert.ok for created-once secrets */
.card.credentials{display:flex;align-items:flex-start;gap:12px;border:1px solid #cfeacb;background:linear-gradient(180deg,#f5fbf4,#fff);color:#14532d}
.card.credentials svg{color:#16a34a;margin-top:3px}
.mono.pill{background:#0b1220;color:#fff;padding:2px 8px;border-radius:6px;font-size:12px}

/* per-database card */
.db-card{padding:14px 16px}
.db-head{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:10px;margin-bottom:10px}
.db-title{font-size:14px;font-weight:700;display:flex;align-items:center;gap:8px;color:#0b1220}
.db-title svg{color:#2563eb;width:16px;height:16px}
table.flat{border:none;border-radius:0;background:transparent;margin-top:4px}
table.flat th{background:transparent;padding:8px 0;border-bottom:1px solid var(--card-line);font-size:10.5px}
table.flat td{padding:10px 0;background:transparent}
table.flat tbody tr:hover{background:transparent}

/* empty state */
.card.empty{display:flex;align-items:center;gap:14px;padding:22px}
.empty-ic{width:42px;height:42px;border-radius:10px;background:#eff5ff;color:#2563eb;display:grid;place-items:center;flex:none}
.empty-ic svg{width:20px;height:20px}

/* tabbed site manager */
.tabs{display:flex;gap:2px;background:#fff;border:1px solid var(--card-line);border-radius:10px;padding:5px;margin-bottom:14px;overflow-x:auto;flex-wrap:wrap;box-shadow:var(--shadow)}
.tabs a{display:inline-flex;align-items:center;gap:6px;padding:7px 12px;border-radius:7px;color:#475569;font-weight:600;font-size:13px;white-space:nowrap;transition:background .12s,color .12s}
.tabs a:hover{background:#f4f6fb;color:#0b1220}
.tabs a.active{background:#0b1220;color:#fff}
.tabs a.active svg{color:#fff;opacity:.9}
.tabs a svg{width:14px;height:14px}

/* toast notifications */
.toasts{position:fixed;top:18px;right:18px;display:flex;flex-direction:column;gap:8px;z-index:9999;pointer-events:none;max-width:calc(100% - 36px)}
.toast{pointer-events:auto;background:#0b1220;color:#fff;padding:11px 14px 11px 14px;border-radius:10px;box-shadow:0 12px 32px rgba(15,23,42,.22);font-weight:600;font-size:13px;max-width:380px;display:flex;align-items:flex-start;gap:10px;transform:translateX(120%);opacity:0;transition:transform .25s cubic-bezier(.2,.7,.2,1.2),opacity .2s ease;border-left:3px solid #3b82f6}
.toast.show{transform:translateX(0);opacity:1}
.toast.ok{border-left-color:#22c55e}
.toast.bad{background:#3a0a0a;border-left-color:#ef4444}
.toast.info{border-left-color:#3b82f6}
.toast .dot{width:8px;height:8px;border-radius:99px;background:#3b82f6;margin-top:5px;flex:none}
.toast.ok .dot{background:#22c55e}
.toast.bad .dot{background:#ef4444}
.toast .msg{flex:1;line-height:1.4;word-break:break-word;white-space:pre-wrap}
.toast .x{background:none;border:none;color:#94a3b8;cursor:pointer;padding:0;margin-left:6px;font-size:14px;line-height:1}
.toast .x:hover{color:#fff}

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
      <a href="/redis" class="{{if eq .View "redis"}}active{{end}}">{{icon "activity"}}<span>Redis Cache</span></a>
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
      <h1>{{.Title}}</h1>
      <div class="right">
        <button type="button" class="searchbtn" onclick="openPalette()" title="Search (Ctrl/Cmd+K)">{{icon "circle"}} <span>Search…</span> <span class="kbd">⌘K</span></button>
        <button type="button" class="iconbtn" id="themeBtn" onclick="toggleTheme()" title="Toggle theme">{{icon "circle"}}</button>
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
      {{else if eq .View "redis"}}{{template "redis" .}}
      {{end}}
    </div>
  </main>
</div>
<div id="toasts" class="toasts" aria-live="polite"></div>

<!-- Cmd+K command palette -->
<div class="modal-bg" id="palette" style="align-items:flex-start;padding-top:12vh">
  <div class="modal palette" role="dialog" aria-label="Quick search">
    <input class="input palette-input" id="paletteInput" placeholder="Search sites, pages, actions…" autocomplete="off">
    <div class="palette-list" id="paletteList"></div>
    <div class="palette-foot"><span><span class="kbd">↑↓</span> navigate</span><span><span class="kbd">↵</span> open</span><span><span class="kbd">esc</span> close</span></div>
  </div>
</div>
<div id="palette-data" hidden>
  <a data-cat="Pages" data-icon="layout" href="/">Dashboard</a>
  <a data-cat="Pages" data-icon="globe" href="/sites">All sites</a>
  <a data-cat="Pages" data-icon="folder" href="/files?path=/">File manager</a>
  <a data-cat="Pages" data-icon="server" href="/services">Services</a>
  <a data-cat="Pages" data-icon="clock" href="/cron">Cron jobs</a>
  <a data-cat="Pages" data-icon="cpu" href="/php">PHP versions</a>
  <a data-cat="Pages" data-icon="activity" href="/redis">Redis cache</a>
  <a data-cat="Pages" data-icon="database" href="/databases">All databases</a>
  <a data-cat="Pages" data-icon="wordpress" href="/wordpress">All WordPress</a>
  <a data-cat="Pages" data-icon="shield" href="/ssl">All certificates</a>
  <a data-cat="Admin" data-icon="users" href="/account">Account</a>
  <a data-cat="Admin" data-icon="activity" href="/audit">Audit log</a>
  <a data-cat="Admin" data-icon="logout" href="/logout">Sign out</a>
  {{range .PaletteSites}}<a data-cat="Sites" data-icon="globe" href="/site?domain={{.Domain}}">{{.Domain}}</a>{{end}}
</div>

<script>
// Theme: dark/light toggle saved per-browser, picks up prefers-color-scheme.
(function(){
  var saved = localStorage.getItem('hostq-theme');
  var sys = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  document.documentElement.dataset.theme = saved || sys;
})();
function toggleTheme(){
  var cur = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
  document.documentElement.dataset.theme = cur;
  localStorage.setItem('hostq-theme', cur);
}

// Cmd+K palette
var paletteItems=[], paletteIndex=0;
function buildPalette(){
  var data=document.getElementById('palette-data');
  if(!data) return;
  paletteItems = Array.prototype.slice.call(data.querySelectorAll('a')).map(function(a){
    return {label:a.textContent.trim(), cat:a.dataset.cat||'', href:a.getAttribute('href'), icon:a.dataset.icon||''};
  });
}
function renderPalette(q){
  q = (q||'').trim().toLowerCase();
  var list = document.getElementById('paletteList');
  list.innerHTML = '';
  var filtered = paletteItems;
  if(q){
    filtered = paletteItems.filter(function(it){ return it.label.toLowerCase().indexOf(q)>=0 || it.cat.toLowerCase().indexOf(q)>=0; });
  }
  if(filtered.length===0){
    list.innerHTML = '<div class="palette-empty">No matches</div>';
    paletteIndex = -1; return;
  }
  paletteIndex = 0;
  var byCat = {};
  filtered.forEach(function(it,i){ (byCat[it.cat] = byCat[it.cat]||[]).push({it:it,i:i}); });
  Object.keys(byCat).forEach(function(cat){
    var h=document.createElement('div'); h.className='palette-cat'; h.textContent=cat; list.appendChild(h);
    byCat[cat].forEach(function(entry){
      var row=document.createElement('a'); row.className='palette-row'; row.href=entry.it.href;
      row.innerHTML='<span class="palette-label">'+escapeHtml(entry.it.label)+'</span><span class="palette-hint">'+escapeHtml(entry.it.cat)+'</span>';
      row.dataset.idx = entry.i;
      list.appendChild(row);
    });
  });
  highlightRow();
}
function escapeHtml(s){ return s.replace(/[&<>"]/g, function(c){ return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]; }); }
function highlightRow(){
  var rows = document.querySelectorAll('#paletteList .palette-row');
  rows.forEach(function(r,i){ r.classList.toggle('on', i===paletteIndex); });
  if(rows[paletteIndex]) rows[paletteIndex].scrollIntoView({block:'nearest'});
}
function openPalette(){
  if(!paletteItems.length) buildPalette();
  var bg=document.getElementById('palette'); bg.classList.add('show');
  var inp=document.getElementById('paletteInput'); inp.value=''; renderPalette('');
  setTimeout(function(){ inp.focus(); }, 30);
}
function closePalette(){ document.getElementById('palette').classList.remove('show'); }
document.addEventListener('keydown',function(e){
  var k=(e.key||'').toLowerCase();
  if((e.ctrlKey||e.metaKey) && k==='k'){ e.preventDefault(); openPalette(); return; }
  var open = document.getElementById('palette') && document.getElementById('palette').classList.contains('show');
  if(!open) return;
  if(k==='escape'){ closePalette(); return; }
  var rows = document.querySelectorAll('#paletteList .palette-row');
  if(k==='arrowdown'){ e.preventDefault(); paletteIndex=Math.min(rows.length-1,paletteIndex+1); highlightRow(); }
  else if(k==='arrowup'){ e.preventDefault(); paletteIndex=Math.max(0,paletteIndex-1); highlightRow(); }
  else if(k==='enter'){ e.preventDefault(); if(rows[paletteIndex]) window.location = rows[paletteIndex].getAttribute('href'); }
});
document.addEventListener('input',function(e){
  if(e.target && e.target.id==='paletteInput') renderPalette(e.target.value);
});
</script>

<script>
// Toast system: also shown for ?output= flash redirects, then URL is cleaned.
function toast(msg, kind){
  var c=document.getElementById('toasts'); if(!c) return;
  var el=document.createElement('div');
  el.className='toast '+(kind||'info');
  var dot=document.createElement('span'); dot.className='dot';
  var body=document.createElement('div'); body.className='msg'; body.textContent=msg;
  var x=document.createElement('button'); x.className='x'; x.type='button'; x.innerHTML='&times;';
  x.addEventListener('click',function(){ dismiss(el); });
  el.appendChild(dot); el.appendChild(body); el.appendChild(x);
  c.appendChild(el);
  requestAnimationFrame(function(){ el.classList.add('show'); });
  setTimeout(function(){ dismiss(el); }, 5000);
}
function dismiss(el){ if(!el) return; el.classList.remove('show'); setTimeout(function(){ if(el.parentNode){ el.parentNode.removeChild(el); } }, 280); }
(function(){
  try{
    var p=new URLSearchParams(location.search);
    var out=p.get('output');
    if(out){
      var l=out.toLowerCase();
      var kind=(l.indexOf('fail')>=0 || l.indexOf('invalid')>=0 || l.indexOf('error')>=0 || l.indexOf('blocked')>=0 || l.indexOf('cannot')>=0) ? 'bad' : 'ok';
      toast(out, kind);
      p.delete('output');
      var q=p.toString();
      history.replaceState({}, '', location.pathname + (q?'?'+q:''));
    }
  }catch(e){}
})();
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
document.addEventListener('keydown',function(e){
  if(e.key==='Escape'){
    document.querySelectorAll('.modal-bg.show').forEach(function(m){ m.classList.remove('show'); });
  }
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
<div class="site-head">
  <div class="site-head-main">
    <a class="site-back" href="/sites" title="All sites">{{icon "chevronUp"}}</a>
    <div>
      <h1>{{.Site.Domain}}</h1>
      <p class="muted mono">{{.Site.Root}}</p>
    </div>
  </div>
  <div class="site-head-meta">
    <span class="badge {{if .Site.Enabled}}ok{{else}}bad{{end}}">{{if .Site.Enabled}}{{icon "check"}} live{{else}}{{icon "x"}} disabled{{end}}</span>
    {{if .Site.SSL}}<span class="badge ok">{{icon "shield"}} SSL</span>{{end}}
    {{if .Site.Cache}}<span class="badge info">cache</span>{{end}}
    <span class="badge">PHP {{.Site.PHPVersion}}</span>
    <a class="btn" href="http{{if .Site.SSL}}s{{end}}://{{.Site.Domain}}" target="_blank">{{icon "globe"}} Visit site</a>
  </div>
</div>

<div class="tabs">
  <a href="/site?domain={{.Site.Domain}}&tab=overview"  class="{{if eq .Tab "overview"}}active{{end}}">{{icon "layout"}} Overview</a>
  <a href="/site?domain={{.Site.Domain}}&tab=database"  class="{{if eq .Tab "database"}}active{{end}}">{{icon "database"}} Database</a>
  <a href="/site?domain={{.Site.Domain}}&tab=wordpress" class="{{if eq .Tab "wordpress"}}active{{end}}">{{icon "wordpress"}} WordPress</a>
  <a href="/site?domain={{.Site.Domain}}&tab=ssl"       class="{{if eq .Tab "ssl"}}active{{end}}">{{icon "shield"}} SSL</a>
  <a href="/site?domain={{.Site.Domain}}&tab=php"       class="{{if eq .Tab "php"}}active{{end}}">{{icon "cpu"}} PHP</a>
  <a href="/files?path=/{{.Site.Domain}}/htdocs">{{icon "folderOpen"}} Files</a>
  <a href="/site?domain={{.Site.Domain}}&tab=security"  class="{{if eq .Tab "security"}}active{{end}}">{{icon "shield"}} Security</a>
  <a href="/site?domain={{.Site.Domain}}&tab=backups"   class="{{if eq .Tab "backups"}}active{{end}}">{{icon "archive"}} Backups</a>
</div>

{{if eq .Tab "overview"}}
  <div class="grid-2">
    <div class="card">
      <h3>Quick actions</h3>
      <div class="actions">
        <a class="btn" href="/files?path=/{{.Site.Domain}}/htdocs">{{icon "folderOpen"}} Files</a>
        <a class="btn" href="/site?domain={{.Site.Domain}}&tab=database">{{icon "database"}} Database</a>
        <a class="btn" href="/site?domain={{.Site.Domain}}&tab=wordpress">{{icon "wordpress"}} WordPress</a>
        <a class="btn" href="/site?domain={{.Site.Domain}}&tab=ssl">{{icon "shield"}} SSL</a>
        <a class="btn" href="/site?domain={{.Site.Domain}}&tab=security">{{icon "shield"}} Security scan</a>
        <a class="btn" href="/site?domain={{.Site.Domain}}&tab=backups">{{icon "archive"}} Backups</a>
      </div>
    </div>
    <div class="card">
      <h3>Site controls</h3>
      <div class="actions">
        <form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Site.Domain}}">
          {{if .Site.Enabled}}<button class="btn" name="action" value="disable">{{icon "power"}} Disable</button>
          {{else}}<button class="btn primary" name="action" value="enable">{{icon "power"}} Enable</button>{{end}}
        </form>
        <form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Site.Domain}}">
          {{if .Site.Cache}}<button class="btn" name="action" value="cache-off">Cache: off</button>
          {{else}}<button class="btn" name="action" value="cache-on">Cache: on</button>{{end}}
        </form>
        <form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Site.Domain}}">
          <button class="btn" name="action" value="permissions">{{icon "shield"}} Fix permissions</button>
        </form>
        <form method="post" action="/site-action" data-confirm="Permanently delete this site, its document root, and Nginx vhost?">
          <input type="hidden" name="domain" value="{{.Site.Domain}}">
          <button class="btn danger" name="action" value="delete">{{icon "trash"}} Delete site</button>
        </form>
      </div>
    </div>
  </div>
{{end}}

{{if eq .Tab "security"}}
  <div class="toolbar">
    <div class="muted">{{if .Scan}}Last scan {{.Scan.When}} · {{.Scan.Scanned}} files · {{.Scan.Took}}{{else}}No scans yet. Run one to inspect this site for malware patterns.{{end}}</div>
    <form method="post" action="/security" style="display:inline">
      <input type="hidden" name="domain" value="{{.Site.Domain}}">
      <button class="btn primary" name="action" value="scan">{{icon "shield"}} Run security scan</button>
    </form>
  </div>
  {{if .Scan}}
    <div class="grid">
      <div class="stat"><div class="label">{{icon "alert"}} Critical</div><div class="val"><span class="bad">{{.Scan.Critical}}</span></div><div class="sub">RCE / known webshell</div></div>
      <div class="stat"><div class="label">{{icon "alert"}} High</div><div class="val">{{.Scan.High}}</div><div class="sub">very suspicious patterns</div></div>
      <div class="stat"><div class="label">{{icon "info"}} Medium</div><div class="val">{{.Scan.Medium}}</div><div class="sub">worth a look</div></div>
      <div class="stat"><div class="label">{{icon "check"}} Scanned</div><div class="val">{{.Scan.Scanned}}</div><div class="sub">files in {{.Scan.Took}}</div></div>
    </div>
    {{if .Scan.Findings}}
    <div class="card" style="padding:0">
      <table>
        <thead><tr><th>Severity</th><th>File</th><th>Rule</th><th>Match</th><th>Modified</th><th class="right-col">Actions</th></tr></thead>
        <tbody>
          {{range .Scan.Findings}}<tr>
            <td>
              {{if eq .Severity "critical"}}<span class="badge bad">{{icon "alert"}} critical</span>
              {{else if eq .Severity "high"}}<span class="badge bad">{{icon "alert"}} high</span>
              {{else if eq .Severity "medium"}}<span class="badge warn">{{icon "info"}} medium</span>
              {{else}}<span class="badge">{{.Severity}}</span>{{end}}
            </td>
            <td class="mono" style="max-width:280px;overflow:hidden;text-overflow:ellipsis">{{.Path}}</td>
            <td><span class="badge">{{.Rule}}</span><div class="muted" style="margin-top:3px;font-size:11.5px">{{.Detail}}</div></td>
            <td class="mono muted" style="font-size:11.5px;max-width:340px;word-break:break-all">{{if .Match}}{{.Match}}{{else}}—{{end}}</td>
            <td class="muted">{{.Modified}}</td>
            <td class="right-col"><div class="actions" style="justify-content:flex-end">
              <form method="post" action="/security" style="display:inline" data-confirm="Quarantine this file? It will be moved to /var/backups/hostq/quarantine/{{$.Site.Domain}}/...">
                <input type="hidden" name="domain" value="{{$.Site.Domain}}">
                <input type="hidden" name="abs" value="{{.AbsPath}}">
                <button class="btn mini" name="action" value="quarantine">{{icon "archive"}} Quarantine</button>
              </form>
              <form method="post" action="/security" style="display:inline" data-confirm="Permanently delete this file?">
                <input type="hidden" name="domain" value="{{$.Site.Domain}}">
                <input type="hidden" name="abs" value="{{.AbsPath}}">
                <button class="btn mini danger" name="action" value="delete">{{icon "trash"}}</button>
              </form>
            </div></td>
          </tr>{{end}}
        </tbody>
      </table>
    </div>
    {{else}}
      <div class="card empty">
        <div class="empty-ic">{{icon "check"}}</div>
        <div><h3 style="margin:0 0 4px;color:var(--ink)">Clean</h3><p class="muted" style="margin:0">No suspicious patterns found in {{.Scan.Scanned}} scanned files. Run again any time after a deploy or plugin install.</p></div>
      </div>
    {{end}}
  {{else}}
    <div class="card empty">
      <div class="empty-ic">{{icon "shield"}}</div>
      <div>
        <h3 style="margin:0 0 4px;color:var(--ink)">No scans yet</h3>
        <p class="muted" style="margin:0">The scanner walks every PHP / HTML / JS / .htaccess file under <span class="mono">{{.Site.Root}}</span> and checks them against a curated set of malware patterns (known webshells, eval-decode, RCE callbacks, hidden iframes, PHP in uploads, etc.).</p>
        <p class="muted" style="margin:6px 0 0">Findings can be <strong>quarantined</strong> (moved to <span class="mono">/var/backups/hostq/quarantine/{{.Site.Domain}}/&lt;timestamp&gt;/</span>) or deleted from this page.</p>
      </div>
    </div>
  {{end}}
{{end}}

{{if eq .Tab "database"}}
  {{if .Created}}<div class="card credentials">{{icon "key"}}
    <div><strong>Database ready</strong> <span class="mono pill">{{.Created}}</span> · user <span class="mono pill">{{.User}}</span> · password <span class="mono pill">{{.Password}}</span><div class="muted" style="font-weight:500;margin-top:4px">Save this password — it is shown only once.</div></div>
  </div>{{end}}
  {{if .DBUser}}<div class="card credentials">{{icon "key"}}
    <div><strong>User ready</strong> <span class="mono pill">{{.DBUser}}</span> · password <span class="mono pill">{{.DBPass}}</span>{{if .DBName}} · on <span class="mono pill">{{.DBName}}</span>{{end}}<div class="muted" style="font-weight:500;margin-top:4px">Shown only once.</div></div>
  </div>{{end}}

  <div class="toolbar">
    <div class="muted">{{len .Databases}} database{{if ne (len .Databases) 1}}s{{end}} on <strong>{{.Site.Domain}}</strong></div>
    <button class="btn primary" onclick="openModal('m-newdb')">{{icon "plus"}} New database</button>
  </div>

  {{range .Databases}}
  <div class="card db-card">
    <div class="db-head">
      <div class="db-title">{{icon "database"}} <span class="mono">{{.Name}}</span></div>
      <div class="actions">
        <a class="btn mini" href="/pma-login?domain={{$.Site.Domain}}&db={{.Name}}&user={{.Name}}" target="_blank">{{icon "terminal"}} Open in phpMyAdmin</a>
        <button class="btn mini" onclick="openAddUser('{{.Name}}')">{{icon "plus"}} Add user</button>
        <form method="post" action="/databases" data-confirm="Drop database {{.Name}}? This deletes all tables and the matching user.">
          <input type="hidden" name="site" value="{{$.Site.Domain}}">
          <input type="hidden" name="target" value="{{.Name}}">
          <button class="btn mini danger" name="action" value="delete">{{icon "trash"}} Drop</button>
        </form>
      </div>
    </div>
    {{if .Users}}<table class="flat">
      <thead><tr><th>User</th><th>Host</th><th class="right-col">Actions</th></tr></thead>
      <tbody>
        {{range .Users}}<tr>
          <td class="mono"><strong>{{.Login}}</strong></td>
          <td class="muted mono">{{.Host}}</td>
          <td class="right-col"><div class="actions" style="justify-content:flex-end">
            <a class="btn mini" href="/pma-login?domain={{$.Site.Domain}}&user={{.Login}}" target="_blank" title="Sign in to phpMyAdmin as this user">{{icon "terminal"}}</a>
            <button class="btn mini" onclick="openResetUser('{{.Login}}')">{{icon "key"}} Reset password</button>
            <form method="post" action="/databases" data-confirm="Drop user {{.Login}} and revoke access?" style="display:inline">
              <input type="hidden" name="site" value="{{$.Site.Domain}}">
              <input type="hidden" name="user" value="{{.Login}}">
              <input type="hidden" name="target" value="{{$.Site.Domain}}">
              <button class="btn mini danger" name="action" value="user-delete">{{icon "trash"}}</button>
            </form>
          </div></td>
        </tr>{{end}}
      </tbody>
    </table>
    {{else}}<p class="muted" style="margin:6px 0 0">No users yet — add one with the button above.</p>{{end}}
  </div>
  {{else}}
    <div class="card empty">
      <div class="empty-ic">{{icon "database"}}</div>
      <div><h3 style="margin:0 0 4px;color:#0b1220">No databases yet</h3><p class="muted" style="margin:0">Click <strong>New database</strong> above. The suffix becomes part of the name, e.g. <span class="mono">{{.DBPrefix}}_main</span>.</p></div>
    </div>
  {{end}}

  <!-- modals -->
  <div class="modal-bg" id="m-newdb"><div class="modal">
    <h3>{{icon "plus"}} New database</h3>
    <p class="muted">Creates a database and a matching user with a 24-char password. Auto-prefixed with <span class="mono">{{.DBPrefix}}_</span>.</p>
    <form method="post" action="/databases">
      <input type="hidden" name="site" value="{{.Site.Domain}}">
      <div class="field"><label>Suffix</label><input class="input mono" name="name" placeholder="main" required autofocus></div>
      <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-newdb')">Cancel</button><button class="btn primary" name="action" value="create">{{icon "check"}} Create</button></div>
    </form>
  </div></div>

  <div class="modal-bg" id="m-adduser"><div class="modal">
    <h3>{{icon "plus"}} Add user</h3>
    <p class="muted">User will receive full privileges on <span class="mono" id="mu-db"></span>.</p>
    <form method="post" action="/databases">
      <input type="hidden" name="site" value="{{.Site.Domain}}">
      <input type="hidden" name="target" id="mu-target">
      <div class="field"><label>Username (letters, digits, _; max 32)</label><input class="input mono" name="user" placeholder="app_user" required></div>
      <div class="field"><label>Password</label><input class="input mono" name="password" type="text" placeholder="leave empty to auto-generate"></div>
      <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-adduser')">Cancel</button><button class="btn primary" name="action" value="user-create">{{icon "check"}} Add user</button></div>
    </form>
  </div></div>

  <div class="modal-bg" id="m-resetuser"><div class="modal">
    <h3>{{icon "key"}} Reset password</h3>
    <p class="muted">Sets a new password for user <span class="mono" id="ru-user-label"></span>.</p>
    <form method="post" action="/databases">
      <input type="hidden" name="site" value="{{.Site.Domain}}">
      <input type="hidden" name="user" id="ru-user">
      <div class="field"><label>New password (8+ chars)</label><input class="input mono" name="password" type="text" minlength="8" required></div>
      <div class="modal-foot"><button type="button" class="btn" onclick="closeModal('m-resetuser')">Cancel</button><button class="btn primary" name="action" value="user-password">{{icon "check"}} Reset</button></div>
    </form>
  </div></div>

  <script>
  function openAddUser(db){ document.getElementById('mu-db').textContent=db; document.getElementById('mu-target').value=db; openModal('m-adduser'); }
  function openResetUser(u){ document.getElementById('ru-user-label').textContent=u; document.getElementById('ru-user').value=u; openModal('m-resetuser'); }
  </script>
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

    <div class="card">
      <div class="toolbar" style="margin:0">
        <div>
          <h3 style="margin:0">{{icon "shield"}} Malfix — file integrity & repair</h3>
          <p class="muted" style="margin:4px 0 0">Verifies WordPress core, plugins, and themes against the official WordPress.org checksums. Reinstall any altered files in place.</p>
        </div>
        <form method="post" action="/malfix" style="display:inline"><input type="hidden" name="domain" value="{{.Site.Domain}}">
          <button class="btn primary" name="action" value="scan">{{icon "refresh"}} Run integrity scan</button>
        </form>
      </div>
      {{if .Malfix}}
        <hr class="sep">
        <div class="muted" style="margin-bottom:10px">Last scan {{.Malfix.When}} · {{.Malfix.Took}} · <strong>{{.Malfix.Summary}}</strong></div>
        {{if and .Malfix.CoreOK (eq (len .Malfix.PluginsFailed) 0) (eq (len .Malfix.ThemesFailed) 0)}}
          <div class="card empty" style="margin:0">
            <div class="empty-ic">{{icon "check"}}</div>
            <div><h3 style="margin:0 0 4px;color:var(--ink)">Clean install</h3><p class="muted" style="margin:0">Every core, plugin and theme file matches its WordPress.org checksum.</p></div>
          </div>
        {{else}}
          {{if or (not .Malfix.CoreOK) (gt (len .Malfix.CoreFailed) 0)}}
          <div class="db-card" style="border:1px solid var(--card-line);border-radius:10px;padding:14px 16px;margin-bottom:10px">
            <div class="db-head" style="margin-bottom:8px">
              <div class="db-title"><span class="badge bad">{{icon "alert"}} core altered</span> <span class="mono" style="font-weight:700">{{len .Malfix.CoreFailed}} file(s)</span></div>
              <form method="post" action="/malfix" data-confirm="Re-download WordPress core and overwrite altered files? wp-content (themes/plugins/uploads) and wp-config.php are kept."><input type="hidden" name="domain" value="{{.Site.Domain}}">
                <button class="btn mini primary" name="action" value="repair-core">{{icon "refresh"}} Repair core</button>
              </form>
            </div>
            {{if .Malfix.CoreFailed}}<ul class="mono" style="margin:0;padding-left:18px;font-size:12px;color:var(--ink-muted)">
              {{range .Malfix.CoreFailed}}<li>{{.}}</li>{{end}}
            </ul>{{end}}
          </div>
          {{end}}
          {{range $slug, $files := .Malfix.PluginsFailed}}
          <div class="db-card" style="border:1px solid var(--card-line);border-radius:10px;padding:14px 16px;margin-bottom:10px">
            <div class="db-head" style="margin-bottom:8px">
              <div class="db-title"><span class="badge bad">{{icon "alert"}} plugin altered</span> <span class="mono" style="font-weight:700">{{$slug}}</span> <span class="muted">— {{len $files}} file(s)</span></div>
              <form method="post" action="/malfix" data-confirm="Re-download plugin {{$slug}} from WordPress.org and overwrite altered files?"><input type="hidden" name="domain" value="{{$.Site.Domain}}"><input type="hidden" name="slug" value="{{$slug}}">
                <button class="btn mini primary" name="action" value="repair-plugin">{{icon "refresh"}} Repair plugin</button>
              </form>
            </div>
            <ul class="mono" style="margin:0;padding-left:18px;font-size:12px;color:var(--ink-muted)">
              {{range $files}}<li>{{.}}</li>{{end}}
            </ul>
          </div>
          {{end}}
          {{range $slug, $files := .Malfix.ThemesFailed}}
          <div class="db-card" style="border:1px solid var(--card-line);border-radius:10px;padding:14px 16px;margin-bottom:10px">
            <div class="db-head" style="margin-bottom:8px">
              <div class="db-title"><span class="badge bad">{{icon "alert"}} theme altered</span> <span class="mono" style="font-weight:700">{{$slug}}</span> <span class="muted">— {{len $files}} file(s)</span></div>
              <form method="post" action="/malfix" data-confirm="Re-download theme {{$slug}} from WordPress.org and overwrite altered files?"><input type="hidden" name="domain" value="{{$.Site.Domain}}"><input type="hidden" name="slug" value="{{$slug}}">
                <button class="btn mini primary" name="action" value="repair-theme">{{icon "refresh"}} Repair theme</button>
              </form>
            </div>
            <ul class="mono" style="margin:0;padding-left:18px;font-size:12px;color:var(--ink-muted)">
              {{range $files}}<li>{{.}}</li>{{end}}
            </ul>
          </div>
          {{end}}
          <form method="post" action="/malfix" data-confirm="Repair core + every altered plugin + every altered theme? Custom (non-WP.org) plugins and themes will be skipped silently — only those known to the WordPress.org repository can be re-downloaded."><input type="hidden" name="domain" value="{{.Site.Domain}}">
            <button class="btn primary" name="action" value="repair-all">{{icon "shield"}} Repair everything</button>
          </form>
        {{end}}
      {{else}}
        <hr class="sep">
        <p class="muted" style="margin:0">No integrity scan yet. The scan runs <span class="mono">wp core verify-checksums</span>, <span class="mono">wp plugin verify-checksums --all</span>, and <span class="mono">wp theme verify-checksums --all</span> and lists every file that no longer matches its published hash.</p>
        <p class="muted" style="margin:6px 0 0">Repair re-downloads the affected component from WordPress.org with <span class="mono">--force</span>. Your <span class="mono">wp-config.php</span>, <span class="mono">wp-content/uploads/</span>, and any non-WP.org code is kept untouched.</p>
      {{end}}
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

{{define "redis"}}
<div class="page-head">
  <div><h1>{{icon "activity"}} Redis</h1><p>Optional in-memory cache. Used by WordPress object-cache plugins when installed.</p></div>
  {{if .Stats.Active}}<span class="badge ok">{{icon "check"}} active</span>{{else}}<span class="badge bad">{{icon "x"}} stopped</span>{{end}}
</div>
{{if .Stats.Active}}
  <div class="grid">
    <div class="stat"><div class="label">{{icon "cpu"}} Used memory</div><div class="val">{{.Stats.UsedMemory}}</div><div class="sub">Peak {{.Stats.PeakMemory}}</div></div>
    <div class="stat"><div class="label">{{icon "box"}} Keys</div><div class="val">{{.Stats.TotalKeys}}</div><div class="sub">db0</div></div>
    <div class="stat"><div class="label">{{icon "users"}} Clients</div><div class="val">{{.Stats.Clients}}</div><div class="sub">connected</div></div>
    <div class="stat"><div class="label">{{icon "activity"}} Ops / sec</div><div class="val">{{.Stats.OpsPerSec}}</div><div class="sub">instantaneous</div></div>
    <div class="stat"><div class="label">{{icon "check"}} Hit rate</div><div class="val">{{.Stats.HitRate}}</div><div class="sub">since boot</div></div>
    <div class="stat"><div class="label">{{icon "clock"}} Uptime</div><div class="val">{{.Stats.UptimeDays}}d</div><div class="sub">v{{.Stats.Version}}</div></div>
  </div>
  <div class="card">
    <h3>Actions</h3>
    <div class="actions">
      <form method="post" action="/redis" data-confirm="Flush ALL Redis keys? This empties every cached object.">
        <button class="btn danger" name="action" value="flush">{{icon "trash"}} Flush all keys</button>
      </form>
      <form method="post" action="/redis">
        <button class="btn" name="action" value="restart">{{icon "refresh"}} Restart Redis</button>
      </form>
      <form method="post" action="/redis">
        <button class="btn" name="action" value="stop">{{icon "stop"}} Stop</button>
      </form>
    </div>
    <p class="muted" style="margin-top:10px">To plug Redis into a WordPress site, install the <span class="mono">redis-cache</span> plugin (or <span class="mono">wp-redis</span>) and enable object cache from its settings. The Redis socket is at <span class="mono">127.0.0.1:6379</span>.</p>
  </div>
{{else}}
  <div class="card empty">
    <div class="empty-ic">{{icon "activity"}}</div>
    <div>
      <h3 style="margin:0 0 4px;color:var(--ink)">Redis is not running</h3>
      <p class="muted" style="margin:0 0 10px">The package is installed but the service is stopped. Start it whenever you need a fast in-memory cache.</p>
      <form method="post" action="/redis"><button class="btn primary" name="action" value="start">{{icon "play"}} Start Redis</button></form>
    </div>
  </div>
{{end}}
{{end}}
`
