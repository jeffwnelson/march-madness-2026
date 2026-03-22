// ============================================================
// State — DATA is loaded via <script src="data/data.js">
// ============================================================
const REGION_NAMES = { 1: 'East', 2: 'South', 3: 'West', 4: 'Midwest' };
let _currentRoundKey = 'r64';
let lbSortMode = 'current';
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

// ============================================================
// Init
// ============================================================
document.getElementById('group-name').textContent = DATA.groupName || '';
renderLeaderboard();
renderBracketPicks();
renderStatusWell();

// Restore state from hash
(function applyHash() {
  const hashParts = location.hash.replace('#', '').split('/');
  const tab = hashParts[0];
  if (tab === 'leaderboard') {
    switchTab('leaderboard', false);
    if (hashParts[1] === 'possible') setLbSort('possible');
    else if (hashParts[1] === 'correct') setLbSort('correct');
  } else if (tab === 'consensus') {
    switchTab('consensus', false);
  }
})();

// ============================================================
// Tab switching
// ============================================================
function switchTab(tab, updateHash) {
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
  document.querySelectorAll('.tab-content').forEach(c => c.classList.toggle('active', c.id === 'tab-' + tab));
  if (updateHash !== false) location.hash = tab;
}

// ============================================================
// Help modal
// ============================================================
function toggleHelp() {
  document.getElementById('help-overlay').classList.toggle('open');
  document.getElementById('help-modal').classList.toggle('open');
  const isOpen = document.getElementById('help-modal').classList.contains('open');
  document.body.style.overflow = isOpen ? 'hidden' : '';
}

// ============================================================
// Leaderboard
// ============================================================
function renderLeaderboard() {
  const container = document.getElementById('leaderboard-content');

  // Build eliminated teams from matchup data
  const eliminatedTeams = new Set();
  Object.values(DATA.rounds).forEach(round => {
    round.matchups.forEach(m => {
      if (m.winnerId) {
        if (m.team1Id && m.team1Id !== m.winnerId) eliminatedTeams.add(m.team1Id);
        if (m.team2Id && m.team2Id !== m.winnerId) eliminatedTeams.add(m.team2Id);
      }
    });
  });

  let html = '<div class="lb-pills">';
  html += `<button class="lb-pill${lbSortMode === 'current' ? ' active' : ''}" onclick="setLbSort('current')">Current Points</button>`;
  html += `<button class="lb-pill${lbSortMode === 'possible' ? ' active' : ''}" onclick="setLbSort('possible')">Possible Points</button>`;
  html += `<button class="lb-pill${lbSortMode === 'correct' ? ' active' : ''}" onclick="setLbSort('correct')">Most Correct</button>`;
  html += '</div>';

  const sorted = [...LEADERBOARD.entries].sort((a, b) => {
    if (lbSortMode === 'current') return b.score - a.score || b.maxPossible - a.maxPossible;
    if (lbSortMode === 'possible') return b.maxPossible - a.maxPossible || b.score - a.score;
    return b.correct - a.correct || a.incorrect - b.incorrect;
  });

  html += '<div class="lb-list">';

  function ordinal(n) {
    const s = ['th','st','nd','rd'];
    const v = n % 100;
    return n + (s[(v - 20) % 10] || s[v] || s[0]);
  }

  sorted.forEach((b, i) => {
    const pts = lbSortMode === 'current' ? b.score : lbSortMode === 'possible' ? b.maxPossible : b.correct;
    const subPts = lbSortMode === 'current' ? `/${b.maxPossible}` : lbSortMode === 'possible' ? `/${b.score}` : `/${b.correct + b.incorrect}`;
    const getPrimary = x => lbSortMode === 'current' ? x.score : lbSortMode === 'possible' ? x.maxPossible : x.correct;
    const rank = (i === 0 || pts !== getPrimary(sorted[i-1]))
      ? ordinal(i + 1) : ordinal(sorted.findIndex(x => getPrimary(x) === pts) + 1);
    const rankNum = sorted.findIndex(x => getPrimary(x) === pts) + 1;
    const placeClass = rankNum === 1 ? ' lb-1st' : rankNum === 2 ? ' lb-2nd' : rankNum === 3 ? ' lb-3rd' : '';
    const champTeam = DATA.teams[b.champion];
    const champOut = eliminatedTeams.has(b.champion);
    const champLogo = champTeam ? `<img src="${champTeam.logo}" alt="${champTeam.abbrev}" loading="lazy" style="width:32px;height:32px;${champOut ? 'filter:grayscale(100%);opacity:0.5;' : ''}">` : '';
    const bIdx = LEADERBOARD.entries.indexOf(b);
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
  location.hash = 'leaderboard/' + mode;
  renderLeaderboard();
}

function openBracketModal(idx) {
  const b = LEADERBOARD.entries[idx];
  if (!b) return;
  const champTeam = DATA.teams[b.champion];
  const name = ENTRY_NAMES[b.entryName] || b.member;
  const pct = Math.round(b.percentile * 100);

  const eliminatedTeams = new Set();
  Object.values(DATA.rounds).forEach(round => {
    round.matchups.forEach(m => {
      if (m.winnerId) {
        if (m.team1Id && m.team1Id !== m.winnerId) eliminatedTeams.add(m.team1Id);
        if (m.team2Id && m.team2Id !== m.winnerId) eliminatedTeams.add(m.team2Id);
      }
    });
  });

  // Final Four display
  const ffOrder = [1, 3, 2, 4];
  const ffSorted = ffOrder.map(region => {
    const tid = (b.finalFour || []).find(id => DATA.teams[id] && DATA.teams[id].region === region);
    return tid ? { ...DATA.teams[tid], id: tid } : null;
  });

  const ffHtml = ffSorted.map(t => {
    if (!t) return '';
    const isOut = eliminatedTeams.has(t.id);
    const borderStyle = isOut ? '2px solid rgba(220,38,38,0.6)' : '1px solid var(--border)';
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

  if (b.eliminated) {
    html += `<div class="bracket-modal-eliminated" style="background:rgba(239,68,68,0.1);color:var(--danger)">Eliminated</div>`;
  }

  html += '<div class="bracket-modal-stats">';
  html += `<div class="bracket-modal-stat"><div class="bracket-modal-stat-value">${b.score}</div><div class="bracket-modal-stat-label">Points</div></div>`;
  html += `<div class="bracket-modal-stat"><div class="bracket-modal-stat-value">${b.maxPossible}</div><div class="bracket-modal-stat-label">Max Possible</div></div>`;
  html += `<div class="bracket-modal-stat"><div class="bracket-modal-stat-value">Top ${pct > 0 ? (100 - pct) + '%' : '—'}</div><div class="bracket-modal-stat-label">National</div></div>`;
  if (b.tiebreaker !== null && b.tiebreaker !== undefined) {
    html += `<div class="bracket-modal-stat"><div class="bracket-modal-stat-value">${Math.round(b.tiebreaker)}</div><div class="bracket-modal-stat-label">Tiebreaker</div></div>`;
  }
  html += '</div>';

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
// Bracket Picks View
// ============================================================
function renderBracketPicks() {
  const container = document.getElementById('consensus-content');

  const rounds = [
    { key: 'r64', label: 'Round of 64' },
    { key: 'r32', label: 'Round of 32' },
    { key: 'sweet16', label: 'Sweet 16' },
    { key: 'elite8', label: 'Elite 8' },
    { key: 'finalFour', label: 'Final Four' },
    { key: 'championship', label: 'Championship' },
  ];

  // Auto-advance pill to first incomplete round
  let defaultRound = 'r64';
  for (const rd of rounds) {
    const roundData = DATA.rounds[rd.key];
    if (roundData && roundData.status === 'complete') {
      const nextIdx = rounds.indexOf(rd) + 1;
      if (nextIdx < rounds.length) defaultRound = rounds[nextIdx].key;
    } else {
      break;
    }
  }

  let html = '<div class="round-tabs">';
  rounds.forEach(rd => {
    html += `<button class="round-tab${rd.key === defaultRound ? ' active' : ''}" data-round="${rd.key}" onclick="selectRound('${rd.key}')">${rd.label}</button>`;
  });
  html += '</div>';
  html += '<div id="round-content"></div>';

  container.innerHTML = html;

  const hashParts = location.hash.replace('#', '').split('/');
  const initRound = (hashParts[0] === 'consensus' && hashParts[1]) ? hashParts[1] : defaultRound;
  selectRound(initRound, false);

  // If hash includes a matchup ID, open it after a short delay
  // (allows the DOM to render the round content first)
  if (hashParts[2]) {
    setTimeout(() => openMatchupDrawer(decodeURIComponent(hashParts[2]), false), 50);
  }
}

// ============================================================
// Round selection
// ============================================================
function selectRound(roundKey, updateHash) {
  _currentRoundKey = roundKey;
  document.querySelectorAll('.round-tab').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.round === roundKey);
  });
  if (updateHash !== false) {
    location.hash = 'consensus/' + roundKey;
  }

  const container = document.getElementById('round-content');
  const roundData = DATA.rounds[roundKey];
  if (!roundData || roundData.matchups.length === 0) {
    container.innerHTML = '<div style="text-align:center;padding:2rem;color:var(--muted)">No matchups yet for this round.</div>';
    return;
  }

  const matchups = [...roundData.matchups].sort((a, b) => a.displayOrder - b.displayOrder);

  let out = '';
  [1, 2, 3, 4].forEach(regionId => {
    const regionMatchups = matchups.filter(m => m.region === regionId);
    if (regionMatchups.length === 0) return;

    let regionLabel = REGION_NAMES[regionId];
    if (roundKey === 'finalFour') {
      regionLabel = regionId <= 2 ? 'Semifinal 1' : 'Semifinal 2';
    } else if (roundKey === 'championship') {
      regionLabel = 'Championship';
    }

    out += `<div class="region-section"><div class="region-section-title">${regionLabel}</div><div class="round-matchups">`;
    regionMatchups.forEach(m => {
      out += renderMatchupCard(m);
    });
    out += '</div></div>';
  });

  container.innerHTML = out;
}

// ============================================================
// Matchup card rendering
// ============================================================
function renderMatchupCard(m) {
  const t1 = DATA.teams[m.team1Id] || {};
  const t2 = DATA.teams[m.team2Id] || {};
  const decided = !!m.winnerId;
  const playing = !decided && (m.status === 'PLAYING' || (m.gameTime && m.gameTime <= Date.now() && m.status !== 'COMPLETE'));
  const hasPicks = m.picks && Object.keys(m.picks).length > 0;

  let matchupCls = 'bk-matchup';
  if (decided) matchupCls += ' decided';
  else if (playing) matchupCls += ' playing';

  let timeStr = '';
  if (decided) {
    timeStr = 'FINAL';
  } else if (playing) {
    timeStr = 'PLAYING';
  } else if (m.gameTime) {
    const d = new Date(m.gameTime);
    timeStr = d.toLocaleString('en-US', { month: 'numeric', day: 'numeric', timeZone: 'America/Chicago' })
      + '<br>' + d.toLocaleString('en-US', { hour: 'numeric', minute: '2-digit', timeZone: 'America/Chicago' });
  }

  // Pick counts for display
  const t1Count = (m.picks && m.picks[m.team1Id]) ? m.picks[m.team1Id].count : 0;
  const t2Count = (m.picks && m.picks[m.team2Id]) ? m.picks[m.team2Id].count : 0;
  const totalPicks = Object.values(m.picks || {}).reduce((sum, p) => sum + p.count, 0);

  const clickAttr = hasPicks ? ` style="cursor:pointer" onclick="openMatchupDrawer(${JSON.stringify(m.id).replace(/"/g, '&quot;')})"` : '';

  let h = `<div class="${matchupCls}"${clickAttr}>`;
  h += '<div class="bk-matchup-inner">';
  h += '<div class="bk-teams-col">';
  h += renderTeamRow(t1, m.team1Id, m.winnerId, decided, t1Count, totalPicks);
  h += renderTeamRow(t2, m.team2Id, m.winnerId, decided, t2Count, totalPicks);
  h += '</div>';
  if (timeStr) {
    h += `<div class="bk-time-col">${timeStr}</div>`;
  }
  h += '</div>';
  h += '</div>';
  return h;
}

function renderTeamRow(team, teamId, winnerId, decided, pickCount, totalPicks) {
  if (!team.name) {
    return `<div class="bk-team" style="opacity:0.2;min-height:24px"><span class="bk-team-name" style="color:var(--muted)">TBD</span></div>`;
  }

  const isWinner = winnerId === teamId;
  const isLoser = decided && winnerId !== teamId;
  let cls = 'bk-team';
  if (isWinner) cls += ' winner';
  if (isLoser) cls += ' loser';

  let countHtml = '';
  if (totalPicks > 0) {
    countHtml = `<span class="bk-team-count">${pickCount}/${totalPicks}</span>`;
  }

  return `<div class="${cls}">
    <img class="bk-team-logo" src="${team.logo}" alt="${team.abbrev}" loading="lazy">
    <span class="bk-team-seed">${team.seed}</span>
    <span class="bk-team-name">${team.name}</span>
    ${countHtml}
  </div>`;
}

// ============================================================
// Matchup drawer (slide-out detail)
// ============================================================
// Store matchups by ID for quick lookup
const _matchupIndex = {};
Object.values(DATA.rounds).forEach(round => {
  round.matchups.forEach(m => {
    _matchupIndex[m.id] = m;
  });
});

function openMatchupDrawer(matchupId, updateHash) {
  const m = _matchupIndex[matchupId];
  if (!m || !m.picks) return;

  if (updateHash !== false) {
    location.hash = 'consensus/' + _currentRoundKey + '/' + encodeURIComponent(matchupId);
  }

  const t1 = DATA.teams[m.team1Id] || {};
  const t2 = DATA.teams[m.team2Id] || {};
  const decided = !!m.winnerId;

  // Build pick groups: team1 picks, team2 picks, neither
  const team1Picks = m.picks[m.team1Id] || { count: 0, entries: [] };
  const team2Picks = m.picks[m.team2Id] || { count: 0, entries: [] };

  const totalPicks = team1Picks.count + team2Picks.count;

  let html = '';

  // Team 1 group
  html += renderPickGroup(t1, m.team1Id, team1Picks, totalPicks, decided && m.winnerId === m.team1Id);

  // Team 2 group
  html += renderPickGroup(t2, m.team2Id, team2Picks, totalPicks, decided && m.winnerId === m.team2Id);

  // Set drawer title
  const title = `${t1.name || '?'} vs ${t2.name || '?'}`;
  document.getElementById('drawer-title').textContent = title;
  document.getElementById('drawer-body').innerHTML = html;
  document.getElementById('drawer-overlay').classList.add('open');
  document.getElementById('drawer').classList.add('open');
  document.body.style.overflow = 'hidden';
}

function renderPickGroup(team, teamId, pickInfo, totalPicks, isWinner) {
  const countColor = isWinner ? 'var(--success)' : 'var(--white)';
  const nameColor = isWinner ? 'var(--success)' : 'var(--white)';

  let html = `<div class="drawer-team-group">
    <div class="drawer-team-header">`;
  if (team.logo) {
    html += `<img src="${team.logo}" alt="${team.abbrev}" style="width:32px;height:32px">`;
  }
  html += `<div class="team-info">
        <div class="team-name" style="color:${nameColor}">(${team.seed}) ${team.name}${isWinner ? ' ✓' : ''}</div>
      </div>
      <div class="pick-count" style="color:${countColor}">${pickInfo.count}/${totalPicks}</div>
    </div>`;

  if (pickInfo.entries.length > 0) {
    pickInfo.entries.forEach(entry => {
      const pLabel = ENTRY_NAMES[entry]
        ? `${escHtml(entry)} <span style="color:var(--muted)">(${escHtml(ENTRY_NAMES[entry])})</span>`
        : escHtml(entry);
      html += `<div class="drawer-picker"><span class="picker-name">${pLabel}</span></div>`;
    });
  } else {
    html += `<div style="color:var(--muted);font-size:0.75rem;padding:0.3rem 0">No one picked this team</div>`;
  }

  html += '</div>';
  return html;
}

function closeDrawer() {
  document.getElementById('drawer-overlay').classList.remove('open');
  document.getElementById('drawer').classList.remove('open');
  document.body.style.overflow = '';
  // Remove matchup ID from hash, keep round
  location.hash = 'consensus/' + _currentRoundKey;
}

// ============================================================
// Status well (help modal)
// ============================================================
function renderStatusWell() {
  const el = document.getElementById('help-status-well');
  if (!el) return;
  const updated = DATA.lastUpdated ? new Date(DATA.lastUpdated) : null;
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
    '<strong>Version:</strong> ' + (DATA.version || 'dev') + '<br>' +
    '<strong>Last Updated:</strong> ' + localTime + ' (' + timeAgo + ')';
}

// ============================================================
// Helpers
// ============================================================
function escHtml(str) {
  const d = document.createElement('div');
  d.textContent = str;
  return d.innerHTML;
}
