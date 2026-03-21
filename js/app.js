// ============================================================
// State
// ============================================================
let LB_DATA = null;
let BP_DATA = null;
const REGION_NAMES = { 1: 'East', 2: 'South', 3: 'West', 4: 'Midwest' };
const ROUND_KEYS = ['r64','r32','sweet16','elite8','finalFour','championship'];
const ROUND_LABELS = { r64:'R64', r32:'R32', sweet16:'Sweet 16', elite8:'Elite 8', finalFour:'Final Four', championship:'Championship' };
const ENTRY_NAMES = {
  "9th Time's The Charm": 'Julie',
  "Eli's Grandpa": 'Tim',
  'Vasquez fam intern👩🏿‍💻': 'Payton',
  'Cornhusker Hustlers 🌽🏀': 'Dylan',
  "John\u2019s Final Fourcast": 'John',
  "Blair N Nelson's Picks 1": 'Blair',
  'Jack Jack Attack 💥': 'Jack',
  "Delka\u2019s Dunk Dynasty": 'Sara',
  'Binkies & Buckets': 'Lane',
  'Frat Rat 🏓 🍻': 'Tommy',
  'Lenny 🐶🐾': 'Lenny',
  "Luke's Hoops": 'Luke',
  "Jeff's Worst Picks Ever": 'Jeff',
  'Nautical Nate ⚓': 'Nathan',
  "M&M's": 'Emery',
  'Swishers Sweet': 'Cheri',
  'Just Rae-ndom ': 'Raelynn',
  'March sadness ft Tressa ': 'Tressa',
  'The Auditor 📈📉📊🧮': 'Andy',
};
let whatIfState = null;
let APP_VERSION = '';

// ============================================================
// Tab switching
// ============================================================
function switchTab(tab, updateHash) {
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
  document.querySelectorAll('.tab-content').forEach(c => c.classList.toggle('active', c.id === 'tab-' + tab));
  if (updateHash !== false) {
    location.hash = tab;
  }
}

// ============================================================
// Data loading
// ============================================================
async function loadData() {
  const [lbRes, bpRes] = await Promise.all([
    fetch('data/leaderboard.json?' + Date.now()),
    fetch('data/bracket-picks.json?' + Date.now())
  ]);
  LB_DATA = await lbRes.json();
  BP_DATA = await bpRes.json();
  document.getElementById('group-name').textContent = LB_DATA.groupName || '';
  fetch('VERSION?' + Date.now()).then(r => r.text()).then(v => {
    APP_VERSION = v.trim();
    renderDataStatus();
  }).catch(() => { renderDataStatus(); });
  renderConsensus();
  renderLeaderboard();
  initWhatIf();
  applyHash();
}

function applyHash() {
  const hash = location.hash.replace('#', '');
  if (!hash) return;
  const parts = hash.split('/');
  const tab = parts[0];
  if (['leaderboard', 'consensus', 'whatif'].includes(tab)) {
    switchTab(tab, false);
    if (tab === 'consensus' && parts[1]) {
      selectRound(parts[1], false);
    }
  }
}

window.addEventListener('hashchange', applyHash);

async function refreshData() {
  const btn = document.getElementById('btnRefresh');
  btn.classList.add('loading');
  btn.disabled = true;
  try {
    await fetch('/api/refresh', { method: 'POST' });
    await loadData();
  } finally {
    btn.classList.remove('loading');
    btn.disabled = false;
  }
}

// ============================================================
// Help: Data Status
// ============================================================
function renderDataStatus() {
  const el = document.getElementById('help-status-well');
  const updated = LB_DATA && LB_DATA.lastUpdated ? new Date(LB_DATA.lastUpdated) : null;

  let timeAgo = '';
  if (updated) {
    const mins = Math.floor((Date.now() - updated.getTime()) / 60000);
    if (mins < 1) timeAgo = 'just now';
    else if (mins < 60) timeAgo = mins + 'm ago';
    else if (mins < 1440) timeAgo = Math.floor(mins / 60) + 'h ' + (mins % 60) + 'm ago';
    else timeAgo = Math.floor(mins / 1440) + 'd ago';
  }

  const localTime = updated
    ? updated.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' })
    : 'Unknown';

  el.innerHTML =
    '<strong>Version:</strong> <span>' + (APP_VERSION || '...') + '</span><br>' +
    '<strong>Last Updated:</strong> <span>' + localTime + '</span> (' + (timeAgo || '...') + ')';
}

// ============================================================
// Helper: build eliminated teams set from BP_DATA
// ============================================================
function buildEliminatedTeams() {
  const eliminatedTeams = new Set();
  Object.values(BP_DATA.rounds).forEach(round => {
    round.matchups.forEach(m => {
      if (m.winner) {
        if (m.team1 && m.team1 !== m.winner) eliminatedTeams.add(m.team1);
        if (m.team2 && m.team2 !== m.winner) eliminatedTeams.add(m.team2);
      }
    });
  });
  return eliminatedTeams;
}

// ============================================================
// Tab 1: Consensus Bracket View
// ============================================================
function renderConsensus() {
  const container = document.getElementById('consensus-content');
  const total = LB_DATA.brackets.length;

  // Round tab selector
  const rounds = [
    { key: 'r64', label: 'Round of 64', pts: '10 pts', games: 32 },
    { key: 'r32', label: 'Round of 32', pts: '20 pts', games: 16 },
    { key: 'sweet16', label: 'Sweet 16', pts: '40 pts', games: 8 },
    { key: 'elite8', label: 'Elite 8', pts: '80 pts', games: 4 },
    { key: 'finalFour', label: 'Final Four', pts: '160 pts', games: 2 },
    { key: 'championship', label: 'Championship', pts: '320 pts', games: 1 },
  ];

  // Determine the current active round using pre-computed round status
  let defaultRound = 'r64';
  for (const rd of rounds) {
    const roundData = BP_DATA.rounds[rd.key];
    if (roundData && roundData.status === 'complete') {
      const nextIdx = rounds.indexOf(rd) + 1;
      if (nextIdx < rounds.length) defaultRound = rounds[nextIdx].key;
    } else {
      break;
    }
  }

  let html = '<div class="round-tabs">';
  rounds.forEach((rd) => {
    html += `<button class="round-tab${rd.key === defaultRound ? ' active' : ''}" data-round="${rd.key}" onclick="selectRound('${rd.key}')">${rd.label}</button>`;
  });
  html += '</div>';

  // Content area for the selected round
  html += '<div id="round-content"></div>';

  container.innerHTML = html;

  // Render the default round
  // Check if hash specifies a round, otherwise use computed default
  const hashParts = location.hash.replace('#', '').split('/');
  const initRound = (hashParts[0] === 'consensus' && hashParts[1]) ? hashParts[1] : defaultRound;
  selectRound(initRound, false);
}

// --- Help modal ---
function toggleHelp() {
  document.getElementById('help-overlay').classList.toggle('open');
  document.getElementById('help-modal').classList.toggle('open');
  const isOpen = document.getElementById('help-modal').classList.contains('open');
  document.body.style.overflow = isOpen ? 'hidden' : '';
}

// --- Drawer for matchup detail ---
let _cardDataStore = [];
function openDrawerByIdx(idx) {
  const d = _cardDataStore[idx];
  if (!d) return;
  openDrawer(d.teamCounts, d.rk, d.winnerId, d.picksData);
}

function openDrawer(teamCounts, roundKey, winnerId, picksData) {
  const total = LB_DATA.brackets.length;
  // Sort: winner first if decided, then by count
  const entries = Object.entries(teamCounts).sort((a, b) => {
    if (winnerId) {
      if (a[0] === winnerId) return -1;
      if (b[0] === winnerId) return 1;
    }
    return b[1] - a[1];
  });

  let html = '';
  entries.forEach(([tid, count]) => {
    const team = BP_DATA.teams[tid];
    if (!team) return;
    const isWinner = winnerId === tid;
    const isConfirmed = isWinner;
    const countColor = isConfirmed ? 'var(--success)' : 'var(--muted)';
    const nameColor = isConfirmed ? 'var(--success)' : 'var(--white)';
    // Get pickers from pre-aggregated picks data
    const pickers = (picksData && picksData[tid]) ? picksData[tid].entries : [];
    html += `<div class="drawer-team-group">
      <div class="drawer-team-header">
        <img src="${team.logo}" alt="${team.abbrev}">
        <div class="team-info">
          <div class="team-name" style="color:${nameColor}">(${team.seed}) ${team.name}${isConfirmed ? ' ✓' : ''}</div>
        </div>
        <div class="pick-count" style="color:${countColor}">${count}/${total}</div>
      </div>`;
    pickers.forEach(entry => {
      html += `<div class="drawer-picker">
        <span class="picker-name">${escHtml(entry)}</span>
      </div>`;
    });
    if (pickers.length === 0) {
      html += `<div style="color:var(--muted);font-size:0.75rem;padding:0.3rem">No one picked this team</div>`;
    }
    html += '</div>';
  });

  // "Neither" section: entries who didn't pick any of the shown teams
  const allPickedEntries = new Set();
  entries.forEach(([tid]) => {
    if (picksData && picksData[tid]) {
      picksData[tid].entries.forEach(e => allPickedEntries.add(e));
    }
  });
  const neitherEntries = LB_DATA.brackets
    .map(b => b.entryName)
    .filter(name => !allPickedEntries.has(name));

  if (neitherEntries.length > 0 && roundKey !== 'r64') {
    html += `<div class="drawer-team-group">
      <div class="drawer-team-header">
        <div class="team-info">
          <div class="team-name" style="color:var(--muted)">None of these teams</div>
        </div>
        <div class="pick-count" style="color:var(--muted)">${neitherEntries.length}/${total}</div>
      </div>`;
    neitherEntries.forEach(entry => {
      html += `<div class="drawer-picker"><span class="picker-name" style="color:var(--muted)">${escHtml(entry)}</span></div>`;
    });
    html += '</div>';
  }

  let drawerTitle;
  if (entries.length >= 2) {
    drawerTitle = `${BP_DATA.teams[entries[0][0]]?.name || '?'} vs ${BP_DATA.teams[entries[1][0]]?.name || '?'}`;
  } else if (entries.length === 1) {
    drawerTitle = BP_DATA.teams[entries[0][0]]?.name || 'Detail';
  } else {
    drawerTitle = 'Detail';
  }
  document.getElementById('drawer-title').textContent = drawerTitle;
  document.getElementById('drawer-body').innerHTML = html;
  document.getElementById('drawer-overlay').classList.add('open');
  document.getElementById('drawer').classList.add('open');
  document.body.style.overflow = 'hidden';
}

function closeDrawer() {
  document.getElementById('drawer-overlay').classList.remove('open');
  document.getElementById('drawer').classList.remove('open');
  document.body.style.overflow = '';
}

// --- Round selection ---
function selectRound(roundKey, updateHash) {
  // Update active tab
  document.querySelectorAll('.round-tab').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.round === roundKey);
  });
  if (updateHash !== false) {
    location.hash = 'consensus/' + roundKey;
  }

  const container = document.getElementById('round-content');
  const total = LB_DATA.brackets.length;
  const roundData = BP_DATA.rounds[roundKey];

  if (!roundData || roundData.matchups.length === 0) {
    container.innerHTML = '<div style="text-align:center;padding:2rem;color:var(--muted)">No matchups decided yet for this round. Check back after more games finish!</div>';
    return;
  }

  const matchups = roundData.matchups.sort((a, b) => a.displayOrder - b.displayOrder);

  // Store card data for drawer lookups (exposed globally)
  _cardDataStore = [];
  const cardDataStore = _cardDataStore;

  // Render a matchup card (clickable to show picker detail)
  function renderCard(matchup) {
    const teamCounts = {};
    const picksData = matchup.picks || {};
    Object.entries(picksData).forEach(([tid, pickInfo]) => {
      teamCounts[tid] = pickInfo.count;
    });

    const entries = Object.entries(teamCounts).sort((a, b) => b[1] - a[1]);
    if (entries.length === 0) return '<div class="bk-empty">TBD</div>';
    const maxCount = entries[0][1];
    const decided = !!matchup.winner;
    const playing = !decided && matchup.gameTime && matchup.gameTime <= Date.now();
    const cardIdx = cardDataStore.length;
    cardDataStore.push({ teamCounts, rk: roundKey, winnerId: matchup.winner, picksData });
    let timeStr = '';
    if (decided) {
      timeStr = 'FINAL';
    } else if (playing) {
      timeStr = 'PLAYING';
    } else if (matchup.gameTime) {
      const d = new Date(matchup.gameTime);
      timeStr = d.toLocaleString('en-US', { month:'numeric', day:'numeric', timeZone:'America/Chicago' }) + '<br>' + d.toLocaleString('en-US', { hour:'numeric', minute:'2-digit', timeZone:'America/Chicago' });
    }
    let matchupCls = 'bk-matchup';
    if (decided) matchupCls += ' decided';
    else if (playing) matchupCls += ' playing';
    let h = `<div class="${matchupCls}" style="cursor:pointer" onclick="openDrawerByIdx(${cardIdx})">`;
    h += '<div class="bk-matchup-inner">';
    h += '<div class="bk-teams-col">';
    const showTeams = entries.slice(0, 2);
    if (matchup.winner && !showTeams.find(e => e[0] === matchup.winner)) {
      const we = entries.find(e => e[0] === matchup.winner);
      if (we) showTeams[1] = we;
    }
    showTeams.forEach(([tid, count]) => {
      const team = BP_DATA.teams[tid];
      if (!team) return;
      const isWinner = matchup.winner === tid;
      const isLoser = decided && matchup.winner !== tid;
      const isFav = !decided && count === maxCount && entries.length > 1;
      let cls = 'bk-team';
      if (isWinner) cls += ' winner';
      if (isLoser) cls += ' loser';
      if (isFav) cls += ' favorite';
      h += `<div class="${cls}">
        <img class="bk-team-logo" src="${team.logo}" alt="${team.abbrev}" loading="lazy">
        <span class="bk-team-seed">${team.seed}</span>
        <span class="bk-team-name">${team.name}</span>
        <span class="bk-team-count">${count}/${total}</span>
      </div>`;
    });
    if (showTeams.length < 2) {
      h += `<div class="bk-team" style="opacity:0.2;min-height:24px"><span class="bk-team-name" style="color:var(--muted)">---</span></div>`;
    }
    h += '</div>'; // close bk-teams-col
    if (timeStr) {
      h += `<div class="bk-time-col">${timeStr}</div>`;
    }
    h += '</div>'; // close bk-matchup-inner
    h += '</div>'; // close bk-matchup
    return h;
  }

  let out = '';

  // Group by region
  [1, 2, 3, 4].forEach(regionId => {
    const regionMatchups = matchups.filter(m => m.region === regionId);
    if (regionMatchups.length === 0) return;
    // For Final Four and Championship, use custom labels
    let regionLabel = REGION_NAMES[regionId];
    if (roundKey === 'finalFour') {
      regionLabel = regionId <= 2 ? 'Semifinal 1 (East/South)' : 'Semifinal 2 (West/Midwest)';
    } else if (roundKey === 'championship') {
      regionLabel = 'Championship';
    }
    out += `<div class="region-section"><div class="region-section-title">${regionLabel}</div><div class="round-matchups">`;
    regionMatchups.forEach(m => {
      out += renderCard(m);
    });
    out += '</div></div>';
  });

  // For rounds where region might be 0 or missing (championship/finalFour)
  const noRegionMatchups = matchups.filter(m => !m.region || m.region === 0);
  if (noRegionMatchups.length > 0) {
    const label = roundKey === 'championship' ? 'Championship' : roundKey === 'finalFour' ? 'Final Four' : '';
    out += `<div class="region-section"><div class="region-section-title">${label}</div><div class="round-matchups">`;
    noRegionMatchups.forEach(m => {
      out += renderCard(m);
    });
    out += '</div></div>';
  }

  container.innerHTML = out;
}

// ============================================================
// Tab: Leaderboard
// ============================================================
let lbSortMode = 'current';

function renderLeaderboard() {
  const container = document.getElementById('leaderboard-content');
  let html = '<div class="lb-pills">';
  html += `<button class="lb-pill${lbSortMode === 'current' ? ' active' : ''}" onclick="setLbSort('current')">Current Points</button>`;
  html += `<button class="lb-pill${lbSortMode === 'possible' ? ' active' : ''}" onclick="setLbSort('possible')">Possible Points</button>`;
  html += '</div>';

  const eliminatedTeams = buildEliminatedTeams();

  const sorted = [...LB_DATA.brackets].sort((a, b) => {
    if (lbSortMode === 'current') return b.score - a.score || b.maxPossible - a.maxPossible;
    return b.maxPossible - a.maxPossible || b.score - a.score;
  });

  html += '<div class="lb-list">';
  function ordinal(n) {
    const s = ['th','st','nd','rd'];
    const v = n % 100;
    return n + (s[(v - 20) % 10] || s[v] || s[0]);
  }

  sorted.forEach((b, i) => {
    const pts = lbSortMode === 'current' ? b.score : b.maxPossible;
    const subPts = lbSortMode === 'current' ? `/${b.maxPossible}` : `/${b.score}`;
    const getPrimary = x => lbSortMode === 'current' ? x.score : x.maxPossible;
    const rank = (i === 0 || pts !== getPrimary(sorted[i-1]))
      ? ordinal(i + 1) : ordinal(sorted.findIndex(x => getPrimary(x) === pts) + 1);
    const rankNum = sorted.findIndex(x => getPrimary(x) === pts) + 1;
    const placeClass = rankNum === 1 ? ' lb-1st' : rankNum === 2 ? ' lb-2nd' : rankNum === 3 ? ' lb-3rd' : '';
    const champTeam = LB_DATA.teams[b.champion];
    const champOut = eliminatedTeams.has(b.champion);
    const champLogo = champTeam ? `<img src="${champTeam.logo}" alt="${champTeam.abbrev}" loading="lazy" style="width:32px;height:32px;${champOut ? 'filter:grayscale(100%);opacity:0.5;' : ''}">` : '';
    const bIdx = LB_DATA.brackets.indexOf(b);
    html += `<div class="lb-row${placeClass}" style="cursor:pointer" onclick="openBracketModal(${bIdx})">
      <div class="lb-rank">${rank}</div>
      ${champLogo ? `<div class="lb-champ">${champLogo}</div>` : ''}
      <div class="lb-info">
        <div class="lb-entry-name">${escHtml(b.entryName)}</div>
        <div class="lb-member">${ENTRY_NAMES[b.entryName] || b.member}</div>
      </div>
      <div class="lb-pts">${pts}<span class="lb-pts-sub">${subPts}</span></div>
    </div>`;
  });
  html += '</div>';

  container.innerHTML = html;
}

function setLbSort(mode) {
  lbSortMode = mode;
  renderLeaderboard();
}

function openBracketModal(idx) {
  const b = LB_DATA.brackets[idx];
  if (!b) return;
  const total = LB_DATA.brackets.length;
  const champTeam = LB_DATA.teams[b.champion];
  const name = ENTRY_NAMES[b.entryName] || b.member;
  const pct = Math.round(b.percentile * 100);

  // Build set of teams eliminated from the tournament
  const eliminatedTeams = buildEliminatedTeams();

  // Final Four teams
  // Sort Final Four by region: East(1) top-left, West(3) top-right, South(2) bottom-left, Midwest(4) bottom-right
  const ffOrder = [1, 3, 2, 4]; // grid order: top-left, top-right, bottom-left, bottom-right
  // Deduplicate finalFour and take up to 4 unique teams
  const ffUnique = [];
  const ffSeen = new Set();
  (b.finalFour || []).forEach(tid => {
    if (!ffSeen.has(tid)) {
      ffSeen.add(tid);
      ffUnique.push(tid);
    }
  });

  const ffSorted = ffOrder.map(region => {
    const tid = ffUnique.find(id => LB_DATA.teams[id] && LB_DATA.teams[id].region === region);
    return tid ? { ...LB_DATA.teams[tid], id: tid } : null;
  });

  // Teams that actually made the Final Four
  const actualFF = new Set();
  const e8Matchups = (BP_DATA.rounds.elite8 && BP_DATA.rounds.elite8.matchups) || [];
  e8Matchups.forEach(m => {
    if (m.winner) actualFF.add(m.winner);
  });
  const ffMatchups = (BP_DATA.rounds.finalFour && BP_DATA.rounds.finalFour.matchups) || [];
  ffMatchups.forEach(m => {
    if (m.team1) actualFF.add(m.team1);
    if (m.team2) actualFF.add(m.team2);
  });

  const ffHtml = ffSorted.map(t => {
    if (!t) return '';
    const isOut = eliminatedTeams.has(t.id);
    const madeIt = actualFF.has(t.id);
    const borderStyle = isOut ? '2px solid rgba(220,38,38,0.6)' : madeIt ? '2px solid rgba(34,197,94,0.6)' : '1px solid var(--border)';
    return `<div style="display:flex;align-items:center;gap:0.5rem;background:var(--bg2);border:${borderStyle};border-radius:10px;padding:0.7rem 0.75rem;width:100%;box-sizing:border-box;position:relative">
      <img src="${t.logo}" alt="${t.abbrev}" loading="lazy" style="width:40px;height:40px;flex-shrink:0;${isOut ? 'filter:grayscale(100%);opacity:0.5;' : ''}">
      <div style="${isOut ? 'opacity:0.5;' : ''}">
        <div style="font-size:0.9rem;font-weight:600;color:var(--text)">${t.name}</div>
        <div style="font-size:0.6rem;color:var(--muted)">${REGION_NAMES[t.region]}</div>
      </div>
      ${isOut ? '<div style="position:absolute;inset:0;display:flex;align-items:center;justify-content:center;font-size:2.5rem;opacity:0.25;pointer-events:none">&#10060;</div>' : ''}
    </div>`;
  }).join('');

  let html = `<div style="font-weight:700;font-size:1.1rem;color:var(--white)">${escHtml(b.entryName)}</div>`;
  html += `<div style="font-size:0.7rem;color:var(--muted);margin-bottom:1rem">${escHtml(name)}</div>`;

  // Champion
  const champEliminated = eliminatedTeams.has(b.champion);
  if (champTeam) {
    html += `<div class="bracket-modal-champ">
      <img src="${champTeam.logo}" alt="${champTeam.abbrev}" loading="lazy" style="${champEliminated ? 'filter:grayscale(100%);opacity:0.5;' : ''}">
      <div class="bracket-modal-champ-info">
        <div class="bracket-modal-champ-name">${champTeam.name}</div>
        <div class="bracket-modal-champ-sub">${champTeam.seed} seed · Champion Pick</div>
      </div>
    </div>`;
  }

  // Bracket status badge
  if (b.eliminated) {
    html += `<div class="bracket-modal-eliminated" style="background:rgba(239,68,68,0.1);color:var(--danger)">Eliminated</div>`;
  }

  // Stats grid
  html += '<div class="bracket-modal-stats">';
  html += `<div class="bracket-modal-stat">
    <div class="bracket-modal-stat-value">${b.score}</div>
    <div class="bracket-modal-stat-label">Points</div>
  </div>`;
  html += `<div class="bracket-modal-stat">
    <div class="bracket-modal-stat-value">${b.maxPossible}</div>
    <div class="bracket-modal-stat-label">Max Possible</div>
  </div>`;
  html += `<div class="bracket-modal-stat">
    <div class="bracket-modal-stat-value">Top ${pct > 0 ? (100 - pct) + '%' : '—'}</div>
    <div class="bracket-modal-stat-label">National</div>
  </div>`;
  if (b.tiebreaker !== null && b.tiebreaker !== undefined) {
    html += `<div class="bracket-modal-stat">
      <div class="bracket-modal-stat-value">${Math.round(b.tiebreaker)}</div>
      <div class="bracket-modal-stat-label">Tiebreaker</div>
    </div>`;
  }
  html += '</div>';

  // Final Four
  if (ffHtml) {
    html += `<div style="margin-top:0.75rem">
      <div style="font-size:0.65rem;color:var(--muted);text-transform:uppercase;letter-spacing:0.5px;margin-bottom:0.35rem;text-align:center">Final Four</div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.35rem">${ffHtml}</div>
    </div>`;
  }

  document.getElementById('bracket-modal-body').innerHTML = html;
  document.getElementById('bracket-modal-overlay').classList.add('open');
  document.getElementById('bracket-modal').classList.add('open');
  document.body.style.overflow = 'hidden';
}

function closeBracketModal() {
  document.getElementById('bracket-modal-overlay').classList.remove('open');
  document.getElementById('bracket-modal').classList.remove('open');
  document.body.style.overflow = '';
}

// ============================================================
// Tab: What-If Simulator (stubbed — needs redesign for new data format)
// ============================================================
function initWhatIf() {
  const container = document.getElementById('whatif-content');
  if (container) {
    container.innerHTML = '<div style="text-align:center;padding:2rem;color:var(--muted)">The What-If Simulator is being redesigned and will return soon.</div>';
  }
}

function resetWhatIf() {
  initWhatIf();
}

function renderWhatIf() {
  initWhatIf();
}

// ============================================================
// Helpers
// ============================================================
function escHtml(str) {
  const d = document.createElement('div');
  d.textContent = str;
  return d.innerHTML;
}

// ============================================================
// Init
// ============================================================
loadData();
