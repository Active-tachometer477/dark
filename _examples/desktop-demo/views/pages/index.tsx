export default function IndexPage({ message, time }: any) {
  return (
    <div>
      <h1>Dark Desktop</h1>
      <p class="subtitle">Go + WebView with bindings, events, and window control</p>

      <div class="section">
        <h2>Go Bindings</h2>
        <p style="color: #aaa; font-size: 13px; margin-bottom: 12px;">
          Call Go functions from JavaScript — they return Promises.
        </p>
        <button onclick="callGreet()">greet("World")</button>
        <button class="secondary" onclick="callAdd()">add(17, 25)</button>
        <button class="secondary" onclick="callServerTime()">server_time()</button>
        <div id="bind-result" class="result">Click a button above...</div>
      </div>

      <div class="section">
        <h2>Events</h2>
        <p style="color: #aaa; font-size: 13px; margin-bottom: 12px;">
          Bidirectional events between Go and JavaScript.
        </p>
        <button onclick="sendEvent()">JS → Go: emit "clicked"</button>
        <button class="secondary" onclick="requestNotify()">Ask Go to emit event</button>
      </div>

      <div class="section">
        <h2>Window Control</h2>
        <p style="color: #aaa; font-size: 13px; margin-bottom: 12px;">
          Control the native window from JavaScript.
        </p>
        <button onclick="dark.setTitle('Title Changed!')">Change Title</button>
        <button class="secondary" onclick="dark.setTitle('Dark Desktop')">Reset Title</button>
        <button class="danger" onclick="dark.close()">Close Window</button>
      </div>

      <div class="section">
        <h2>Event Log</h2>
        <div id="log">
          <div class="entry" style="color: #555;">Waiting for events...</div>
        </div>
      </div>

      <script dangerouslySetInnerHTML={{ __html: `
        var logEl = document.getElementById('log');
        var resultEl = document.getElementById('bind-result');
        var first = true;

        function log(cls, msg) {
          if (first) { logEl.innerHTML = ''; first = false; }
          var d = document.createElement('div');
          d.className = 'entry ' + cls;
          var ts = new Date().toLocaleTimeString();
          d.textContent = '[' + ts + '] ' + msg;
          logEl.appendChild(d);
          logEl.scrollTop = logEl.scrollHeight;
        }

        // --- Bindings ---
        async function callGreet() {
          var result = await greet('World');
          resultEl.textContent = 'greet("World") => ' + JSON.stringify(result);
          log('go', 'greet("World") => ' + result);
        }

        async function callAdd() {
          var result = await add(17, 25);
          resultEl.textContent = 'add(17, 25) => ' + result;
          log('go', 'add(17, 25) => ' + result);
        }

        async function callServerTime() {
          var result = await server_time();
          resultEl.textContent = 'server_time() => ' + result;
          log('go', 'server_time() => ' + result);
        }

        // --- Events ---
        dark.on('notify', function(data) {
          log('event', 'Received from Go: ' + JSON.stringify(data));
        });

        dark.on('counter', function(data) {
          log('event', 'Go counter: ' + data.count);
        });

        function sendEvent() {
          dark.emit('clicked', { x: Math.round(Math.random() * 100), y: Math.round(Math.random() * 100) });
          log('js', 'Emitted "clicked" event to Go');
        }

        async function requestNotify() {
          await request_notify();
          log('js', 'Asked Go to send a notification event');
        }

        log('js', 'Desktop app ready');
      `}} />
    </div>
  );
}
