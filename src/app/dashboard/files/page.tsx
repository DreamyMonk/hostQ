'use client';
import { useEffect, useState, useCallback } from 'react';
import {
  ChevronRight, Upload, Plus, Trash2,
  Edit3, Save, RefreshCw, Folder, FileText,
  FileCode, Image as ImageIcon, ArrowLeft
} from 'lucide-react';

interface FileItem {
  name: string;
  type: 'dir' | 'file';
  size: string;
  sizeBytes: number;
  modified: string;
  ext: string;
}

interface DirData {
  items: FileItem[];
  path: string;
  breadcrumbs: { name: string; path: string }[];
}

function getFileIcon(item: FileItem) {
  if (item.type === 'dir') return <Folder size={16} color="#f59e0b" />;
  const codeExts = ['php','js','ts','tsx','jsx','html','css','py','sh','json','xml','yaml','yml'];
  const imgExts  = ['jpg','jpeg','png','gif','svg','webp','ico'];
  if (codeExts.includes(item.ext)) return <FileCode size={16} color="#3b82f6" />;
  if (imgExts.includes(item.ext))  return <ImageIcon size={16} color="#22c55e" />;
  return <FileText size={16} color="#8b949e" />;
}

export default function FilesPage() {
  const [currentPath, setCurrentPath] = useState('/');
  const [dirData, setDirData]         = useState<DirData | null>(null);
  const [loading, setLoading]         = useState(true);
  const [selected, setSelected]       = useState<string[]>([]);

  // Editor
  const [editorOpen, setEditorOpen]   = useState(false);
  const [editorFile, setEditorFile]   = useState('');
  const [editorPath, setEditorPath]   = useState('');
  const [editorContent, setEditorContent] = useState('');
  const [editorSaving, setEditorSaving]   = useState(false);

  // Dialogs
  const [newFolderName, setNewFolderName] = useState('');
  const [newFileName, setNewFileName]     = useState('');
  const [renameTarget, setRenameTarget]   = useState('');
  const [renameTo, setRenameTo]           = useState('');
  const [showNewFolder, setShowNewFolder] = useState(false);
  const [showNewFile, setShowNewFile]     = useState(false);
  const [showRename, setShowRename]       = useState(false);
  const [msg, setMsg] = useState<{ type: 'success'|'error'; text: string } | null>(null);
  const [uploading, setUploading] = useState(false);

  const showMsg = (type: 'success'|'error', text: string) => {
    setMsg({ type, text });
    setTimeout(() => setMsg(null), 4000);
  };

  const loadDir = useCallback(async (p: string) => {
    setLoading(true);
    setSelected([]);
    try {
      const r = await fetch(`/api/files?path=${encodeURIComponent(p)}&action=list`);
      const d = await r.json();
      if (d.error) { showMsg('error', d.error); }
      else { setDirData(d); setCurrentPath(p); }
    } finally { setLoading(false); }
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const startPath = params.get('path') || '/';
    const id = setTimeout(() => { void loadDir(startPath); }, 0);
    return () => clearTimeout(id);
  }, [loadDir]);

  const openFile = async (item: FileItem, itemPath: string) => {
    if (item.type === 'dir') { loadDir(itemPath); return; }
    const r = await fetch(`/api/files?path=${encodeURIComponent(itemPath)}&action=read`);
    const d = await r.json();
    if (d.error) { showMsg('error', d.error); return; }
    setEditorFile(item.name);
    setEditorPath(itemPath);
    setEditorContent(d.content);
    setEditorOpen(true);
  };

  const saveFile = async () => {
    setEditorSaving(true);
    try {
      const r = await fetch('/api/files', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'save', path: editorPath, content: editorContent }),
      });
      const d = await r.json();
      if (d.success) showMsg('success', 'File saved!');
      else showMsg('error', d.error);
    } finally { setEditorSaving(false); }
  };

  const createFolder = async () => {
    if (!newFolderName.trim()) return;
    const r = await fetch('/api/files', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'mkdir', path: currentPath, name: newFolderName }),
    });
    const d = await r.json();
    if (d.success) { showMsg('success', d.message); loadDir(currentPath); setShowNewFolder(false); setNewFolderName(''); }
    else showMsg('error', d.error);
  };

  const createFile = async () => {
    if (!newFileName.trim()) return;
    const r = await fetch('/api/files', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'create_file', path: currentPath, name: newFileName, content: '' }),
    });
    const d = await r.json();
    if (d.success) { showMsg('success', d.message); loadDir(currentPath); setShowNewFile(false); setNewFileName(''); }
    else showMsg('error', d.error);
  };

  const renameItem = async () => {
    if (!renameTo.trim()) return;
    const r = await fetch('/api/files', {
      method: 'PATCH', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: renameTarget, newName: renameTo }),
    });
    const d = await r.json();
    if (d.success) { showMsg('success', d.message); loadDir(currentPath); setShowRename(false); setRenameTo(''); }
    else showMsg('error', d.error);
  };

  const deleteItems = async () => {
    if (!selected.length || !confirm(`Delete ${selected.length} item(s)?`)) return;
    for (const p of selected) {
      await fetch('/api/files', {
        method: 'DELETE', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: p }),
      });
    }
    showMsg('success', `Deleted ${selected.length} item(s)`);
    loadDir(currentPath);
  };

  const uploadFile = async (file: File | null) => {
    if (!file) return;
    setUploading(true);
    try {
      const form = new FormData();
      form.set('path', currentPath);
      form.set('file', file);
      const r = await fetch('/api/files', { method: 'POST', body: form });
      const d = await r.json();
      if (d.success) { showMsg('success', d.message); loadDir(currentPath); }
      else showMsg('error', d.error);
    } finally {
      setUploading(false);
    }
  };

  const toggleSelect = (p: string) => {
    setSelected(s => s.includes(p) ? s.filter(x => x !== p) : [...s, p]);
  };

  const itemPath = (item: FileItem) => {
    const base = currentPath === '/' ? '' : currentPath;
    return `${base}/${item.name}`;
  };

  if (editorOpen) {
    return (
      <div className="fade-in">
        <div style={{ display:'flex', alignItems:'center', justifyContent:'space-between', marginBottom:16 }}>
          <div style={{ display:'flex', alignItems:'center', gap:10 }}>
            <button onClick={() => setEditorOpen(false)} className="btn btn-ghost btn-sm"><ArrowLeft size={14}/>Back</button>
            <span style={{ fontSize:14, fontWeight:600 }}>{editorFile}</span>
            <span className="mono" style={{ fontSize:11, color:'var(--text-muted)' }}>{editorPath}</span>
          </div>
          <button id="save-file-btn" onClick={saveFile} className="btn btn-primary btn-sm" disabled={editorSaving}>
            <Save size={14}/>{editorSaving ? 'Saving…' : 'Save File'}
          </button>
        </div>
        {msg && <div className={`alert ${msg.type === 'success' ? 'alert-success' : 'alert-error'}`}>{msg.text}</div>}
        <div className="glass-card" style={{ overflow:'hidden' }}>
          <textarea
            id="file-editor"
            value={editorContent}
            onChange={e => setEditorContent(e.target.value)}
            spellCheck={false}
            style={{
              width:'100%', minHeight:'65vh', background:'#0d1117', color:'#e6edf3',
              border:'none', outline:'none', padding:20, resize:'vertical',
              fontFamily:"'JetBrains Mono', 'Fira Code', monospace", fontSize:13, lineHeight:1.6,
            }}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="fade-in">
      {/* Header */}
      <div className="page-header" style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div>
          <h1 className="page-title">File Manager</h1>
          <p className="page-subtitle">Browse, upload, edit, rename, and delete files inside the configured server root</p>
        </div>
        <div style={{ display:'flex', gap:8 }}>
          {selected.length > 0 && (
            <button id="delete-selected-btn" onClick={deleteItems} className="btn btn-danger btn-sm">
              <Trash2 size={14}/>Delete ({selected.length})
            </button>
          )}
          <button id="new-folder-btn" onClick={() => setShowNewFolder(true)} className="btn btn-ghost btn-sm"><Plus size={14}/>New Folder</button>
          <button id="new-file-btn" onClick={() => setShowNewFile(true)} className="btn btn-ghost btn-sm"><FileText size={14}/>New File</button>
          <label className="btn btn-ghost btn-sm" style={{ cursor:'pointer' }}>
            <Upload size={14}/>{uploading ? 'Uploading...' : 'Upload'}
            <input type="file" style={{ display:'none' }} onChange={e => uploadFile(e.target.files?.[0] || null)} />
          </label>
          <button id="refresh-files-btn" onClick={() => loadDir(currentPath)} className="btn btn-ghost btn-sm">
            <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }}/>
          </button>
        </div>
      </div>

      {msg && <div className={`alert ${msg.type === 'success' ? 'alert-success' : 'alert-error'}`}>{msg.text}</div>}

      {/* Breadcrumbs */}
      {dirData && (
        <div style={{ display:'flex', alignItems:'center', gap:6, marginBottom:16, fontSize:13, flexWrap:'wrap' }}>
          {dirData.breadcrumbs.map((bc, i) => (
            <span key={bc.path} style={{ display:'flex', alignItems:'center', gap:4 }}>
              {i > 0 && <ChevronRight size={12} color="var(--text-muted)" />}
              <button onClick={() => loadDir(bc.path)} style={{
                background:'none', border:'none', cursor:'pointer', padding:'2px 6px', borderRadius:4,
                color: i === dirData.breadcrumbs.length - 1 ? 'var(--text-primary)' : 'var(--text-secondary)',
                fontWeight: i === dirData.breadcrumbs.length - 1 ? 600 : 400, fontSize:13
              }}>
                {bc.name}
              </button>
            </span>
          ))}
        </div>
      )}

      {/* File table */}
      <div className="glass-card">
        {loading ? (
          <div style={{ padding:40, textAlign:'center', color:'var(--text-muted)' }}>
            <div className="spinner" style={{ margin:'0 auto 10px' }} />Loading…
          </div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th style={{ width:30 }}><input type="checkbox" onChange={e => e.target.checked ? setSelected(dirData?.items.map(it => itemPath(it)) || []) : setSelected([])} /></th>
                <th>Name</th><th>Size</th><th>Modified</th><th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {currentPath !== '/' && (
                <tr>
                  <td />
                  <td>
                    <button onClick={() => {
                      const parts = currentPath.split('/').filter(Boolean);
                      parts.pop();
                      loadDir('/' + parts.join('/') || '/');
                    }} style={{ display:'flex', alignItems:'center', gap:8, background:'none', border:'none', cursor:'pointer', color:'var(--text-secondary)', fontSize:13 }}>
                      <ArrowLeft size={14} /> ..
                    </button>
                  </td>
                  <td colSpan={3} />
                </tr>
              )}
              {dirData?.items.length === 0 && (
                <tr><td colSpan={5} style={{ textAlign:'center', padding:30, color:'var(--text-muted)' }}>Empty directory</td></tr>
              )}
              {dirData?.items.map(item => {
                const p = itemPath(item);
                const isSel = selected.includes(p);
                return (
                  <tr key={item.name} style={{ background: isSel ? 'rgba(59,130,246,0.05)' : '' }}>
                    <td><input type="checkbox" checked={isSel} onChange={() => toggleSelect(p)} /></td>
                    <td>
                      <button onClick={() => openFile(item, p)} style={{
                        display:'flex', alignItems:'center', gap:8, background:'none', border:'none',
                        cursor:'pointer', color:'var(--text-primary)', fontSize:13, textAlign:'left'
                      }}>
                        {getFileIcon(item)}
                        {item.name}
                      </button>
                    </td>
                    <td style={{ color:'var(--text-muted)', fontSize:12 }}>{item.size || '—'}</td>
                    <td style={{ color:'var(--text-muted)', fontSize:12 }}>
                      {new Date(item.modified).toLocaleDateString()}
                    </td>
                    <td>
                      <div style={{ display:'flex', gap:4 }}>
                        <button id={`rename-${item.name}`} title="Rename" onClick={() => {
                          setRenameTarget(p); setRenameTo(item.name); setShowRename(true);
                        }} className="btn btn-ghost btn-sm" style={{ padding:'4px 8px' }}>
                          <Edit3 size={12}/>
                        </button>
                        <button id={`delete-${item.name}`} title="Delete" onClick={async () => {
                          if (!confirm(`Delete "${item.name}"?`)) return;
                          await fetch('/api/files', {
                            method:'DELETE', headers:{'Content-Type':'application/json'},
                            body: JSON.stringify({ path: p }),
                          });
                          showMsg('success', `Deleted "${item.name}"`);
                          loadDir(currentPath);
                        }} className="btn btn-danger btn-sm" style={{ padding:'4px 8px' }}>
                          <Trash2 size={12}/>
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Dialogs */}
      {showNewFolder && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.7)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:1000 }}>
          <div className="glass-card" style={{ padding:24, width:360 }}>
            <div style={{ fontWeight:700, marginBottom:16 }}>New Folder</div>
            <input id="new-folder-input" className="input" value={newFolderName} onChange={e => setNewFolderName(e.target.value)} placeholder="folder-name" autoFocus onKeyDown={e => e.key === 'Enter' && createFolder()} />
            <div style={{ display:'flex', gap:8, marginTop:14, justifyContent:'flex-end' }}>
              <button onClick={() => setShowNewFolder(false)} className="btn btn-ghost btn-sm">Cancel</button>
              <button id="create-folder-btn" onClick={createFolder} className="btn btn-primary btn-sm">Create</button>
            </div>
          </div>
        </div>
      )}

      {showNewFile && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.7)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:1000 }}>
          <div className="glass-card" style={{ padding:24, width:360 }}>
            <div style={{ fontWeight:700, marginBottom:16 }}>New File</div>
            <input id="new-file-input" className="input" value={newFileName} onChange={e => setNewFileName(e.target.value)} placeholder="filename.txt" autoFocus onKeyDown={e => e.key === 'Enter' && createFile()} />
            <div style={{ display:'flex', gap:8, marginTop:14, justifyContent:'flex-end' }}>
              <button onClick={() => setShowNewFile(false)} className="btn btn-ghost btn-sm">Cancel</button>
              <button id="create-file-btn" onClick={createFile} className="btn btn-primary btn-sm">Create</button>
            </div>
          </div>
        </div>
      )}

      {showRename && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.7)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:1000 }}>
          <div className="glass-card" style={{ padding:24, width:360 }}>
            <div style={{ fontWeight:700, marginBottom:16 }}>Rename</div>
            <input id="rename-input" className="input" value={renameTo} onChange={e => setRenameTo(e.target.value)} autoFocus onKeyDown={e => e.key === 'Enter' && renameItem()} />
            <div style={{ display:'flex', gap:8, marginTop:14, justifyContent:'flex-end' }}>
              <button onClick={() => setShowRename(false)} className="btn btn-ghost btn-sm">Cancel</button>
              <button id="rename-confirm-btn" onClick={renameItem} className="btn btn-primary btn-sm">Rename</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
