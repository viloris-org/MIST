function el(id) { return document.getElementById(id); }

function fmtRound(n) {
  if (n >= 1e9) return (n / 1e9).toFixed(1) + 'G';
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M';
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K';
  return n.toString();
}

function fmtBytes(n) {
  if (n >= 1e9) return (n / 1e9).toFixed(2) + ' GB';
  if (n >= 1e6) return (n / 1e6).toFixed(2) + ' MB';
  if (n >= 1e3) return (n / 1e3).toFixed(2) + ' KB';
  return n + ' B';
}

function fmtDuration(seconds) {
  var d = Math.floor(seconds / 86400);
  var h = Math.floor((seconds % 86400) / 3600);
  var m = Math.floor((seconds % 3600) / 60);
  var s = Math.floor(seconds % 60);
  var parts = [];
  if (d > 0) parts.push(d + 'd');
  if (h > 0) parts.push(h + 'h');
  if (m > 0) parts.push(m + 'm');
  if (parts.length === 0 || s > 0) parts.push(s + 's');
  return parts.join(' ') || '0s';
}

function update(data) {
  // Server state
  var state = data.server_state || 'unknown';
  el('server-state').textContent = state;
  var dot = document.querySelector('.dot');
  if (state === 'running') {
    dot.className = 'dot';
  } else {
    dot.className = 'dot error';
  }

  el('server-addr').textContent = data.server || '--';

  // Uptime
  if (data.started_at) {
    var started = new Date(data.started_at);
    var elapsed = (Date.now() - started.getTime()) / 1000;
    el('uptime').textContent = 'up ' + fmtDuration(elapsed);
  }

  // Sessions
  var sp = data.client ? data.client.session_pool : {};
  el('active-sessions').textContent = sp.sessions || 0;
  el('idle-sessions').textContent = (sp.idle_sessions || 0) + ' idle';
  el('open-streams').textContent = sp.open_streams || 0;
  el('total-streams').textContent = fmtRound(sp.cumulative_streams || 0) + ' total';
  el('total-sessions').textContent = fmtRound(sp.cumulative_sessions || 0);

  // Connections
  el('active-conns').textContent = data.active_connections || 0;
  el('accepted-conns').textContent = fmtRound(data.accepted_connections || 0) + ' accepted';

  // TUN
  if (data.tun) {
    el('tun-section').style.display = 'block';
    el('tun-name').textContent = data.tun.name || '--';
    el('tun-addr').textContent = data.tun.address || '--';
    el('tun-rx-packets').textContent = fmtRound(data.tun.rx_packets || 0);
    el('tun-rx-bytes').textContent = fmtBytes(data.tun.rx_bytes || 0);
    el('tun-tx-packets').textContent = fmtRound(data.tun.tx_packets || 0);
    el('tun-tx-bytes').textContent = fmtBytes(data.tun.tx_bytes || 0);
  }

  // Version
  if (data.version) {
    el('version').textContent = 'mist/' + data.version;
  }
}

async function refresh() {
  try {
    var resp = await fetch('/api/status');
    if (resp.ok) {
      var data = await resp.json();
      update(data);
    }
  } catch (e) {
    // Dashboard might not be ready yet.
  }
}

refresh();
setInterval(refresh, 2000);
