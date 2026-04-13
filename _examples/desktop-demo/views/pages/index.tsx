import { island } from 'dark';
import Clock from '../islands/clock.tsx';

const LiveClock = island('clock', Clock);

export default function IndexPage({ notes, username, flashes, _errors, _formData }: any) {
  const getError = (field: string) => _errors?.find((e: any) => e.field === field);
  const titleErr = getError('title');

  return (
    <div id="app">
      <div class="header">
        <h1>Dark Notes</h1>
        <div class="header-right">
          {username
            ? <span class="user-badge">{username}</span>
            : <form method="POST" action="/username" class="username-form">
                <input name="username" placeholder="Your name" />
                <button class="btn-sm">Set</button>
              </form>
          }
        </div>
      </div>

      {flashes?.notice && <div class="flash">{flashes.notice}</div>}

      {/* Island: live clock with client-side state */}
      <div class="card">
        <h3>Island Component</h3>
        <LiveClock label="Client-side hydrated clock" />
      </div>

      {/* Desktop bindings + events */}
      <div class="card">
        <h3>Desktop Bridge</h3>
        <div class="actions">
          <button class="btn-ghost btn-sm" onclick="doExport()">Export Notes</button>
          <button class="btn-ghost btn-sm" onclick="showSystemInfo()">System Info</button>
          <button class="btn-ghost btn-sm" onclick="dark.setTitle('Dark Notes — ' + new Date().toLocaleTimeString())">Update Title</button>
        </div>
        <div id="bridge-output" style="margin-top: 8px; display: none;"></div>
      </div>

      {/* Native OS features */}
      <div class="card">
        <h3>Native Features</h3>
        <div class="actions">
          <button class="btn-ghost btn-sm" onclick="doOpenFile()">Open File</button>
          <button class="btn-ghost btn-sm" onclick="doSaveFile()">Save File</button>
          <button class="btn-ghost btn-sm" onclick="doPickFolder()">Pick Folder</button>
          <button class="btn-ghost btn-sm" onclick="doClipboardRead()">Read Clipboard</button>
          <button class="btn-ghost btn-sm" onclick="doClipboardWrite()">Copy "Hello"</button>
          <button class="btn-ghost btn-sm" onclick="doNotify()">OS Notification</button>
        </div>
        <div id="native-output" style="margin-top: 8px; display: none;"></div>
      </div>

      {/* External links (should open in system browser) */}
      <div class="card">
        <h3>External Links</h3>
        <div class="actions">
          <a href="https://github.com/i2y/dark" class="btn-ghost btn-sm">GitHub Repo</a>
          <a href="https://htmx.org" class="btn-ghost btn-sm">htmx.org</a>
          <button class="btn-ghost btn-sm" onclick="dark.openExternal('https://preactjs.com')">Preact (programmatic)</button>
        </div>
      </div>

      {/* Note form with validation (htmx) */}
      <div class="card">
        <h3>Add Note</h3>
        <form hx-post="/notes" hx-target="#app" hx-swap="outerHTML">
          <div class={`form-row ${titleErr ? 'has-error' : ''}`}>
            <label>Title</label>
            <input name="title" value={_formData?.title || ''} placeholder="Note title..." />
            {titleErr && <div class="field-error">{titleErr.message}</div>}
          </div>
          <div class="form-row">
            <label>Body</label>
            <textarea name="body" placeholder="Optional body text...">{_formData?.body || ''}</textarea>
          </div>
          <button type="submit" style="margin-top: 4px;">Add Note</button>
        </form>
      </div>

      {/* Notes list (htmx delete) */}
      <div class="card">
        <h3>Notes ({notes?.length || 0})</h3>
        {(!notes || notes.length === 0)
          ? <div class="empty">No notes yet. Add one above!</div>
          : notes.map((note: any) => (
              <div class="note" key={note.id}>
                <div>
                  <div class="note-title">{note.title}</div>
                  {note.body && <div class="note-body">{note.body}</div>}
                  <div class="note-time">{note.createdAt}</div>
                </div>
                <button
                  class="note-delete"
                  hx-delete={`/notes/${note.id}`}
                  hx-target="#app"
                  hx-swap="outerHTML"
                  title="Delete"
                >&times;</button>
              </div>
            ))
        }
      </div>

      {/* Toast container for desktop events */}
      <div id="toast-container"></div>

      <script dangerouslySetInnerHTML={{ __html: `
        // --- Desktop Bindings ---
        async function doExport() {
          try {
            var path = await export_notes();
            showOutput('Exported to: ' + path);
          } catch(e) {
            showOutput('Export failed: ' + e.message);
          }
        }

        async function showSystemInfo() {
          var info = await system_info();
          var el = document.getElementById('bridge-output');
          el.style.display = 'block';
          el.innerHTML = '<div class="info-grid">'
            + infoItem('Hostname', info.hostname)
            + infoItem('OS', info.os)
            + infoItem('Arch', info.arch)
            + infoItem('CPUs', info.cpus)
            + infoItem('Go', info.goVersion)
            + infoItem('Home', info.homeDir)
            + '</div>';
        }

        function infoItem(label, value) {
          return '<div class="info-item"><span class="info-label">' + label + '</span> <span class="info-value">' + value + '</span></div>';
        }

        function showOutput(msg) {
          var el = document.getElementById('bridge-output');
          el.style.display = 'block';
          el.innerHTML = '<div class="info-item"><span class="info-value">' + msg + '</span></div>';
        }

        // --- Native Features ---
        function showNative(msg) {
          var el = document.getElementById('native-output');
          el.style.display = 'block';
          el.innerHTML = '<div class="info-item"><span class="info-value">' + msg + '</span></div>';
        }

        async function doOpenFile() {
          var path = await dark.openFile({ title: 'Open a file', filters: ['*.txt', '*.md', '*.json'] });
          showNative(path ? 'Opened: ' + path : 'Cancelled');
        }

        async function doSaveFile() {
          var path = await dark.saveFile({ title: 'Save as', filename: 'notes.json' });
          showNative(path ? 'Save to: ' + path : 'Cancelled');
        }

        async function doPickFolder() {
          var path = await dark.pickFolder({ title: 'Select folder' });
          showNative(path ? 'Folder: ' + path : 'Cancelled');
        }

        async function doClipboardRead() {
          var text = await dark.readClipboard();
          showNative('Clipboard: ' + (text || '(empty)'));
        }

        async function doClipboardWrite() {
          await dark.writeClipboard('Hello from Dark Desktop!');
          showNative('Copied to clipboard!');
        }

        async function doNotify() {
          await dark.notify('Dark Notes', 'This is a native OS notification!');
        }

        // --- Desktop Events ---
        dark.on('notification', function(data) {
          showToast(data.message);
        });

        function showToast(msg) {
          var toast = document.createElement('div');
          toast.className = 'toast';
          toast.textContent = msg;
          document.body.appendChild(toast);
          setTimeout(function() { toast.remove(); }, 3000);
        }

        // Update window title with note count
        var noteCount = ${notes?.length || 0};
        dark.emit('update-title', { title: 'Dark Notes (' + noteCount + ')' });
      `}} />
    </div>
  );
}
