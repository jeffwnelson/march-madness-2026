// ============================================================
// State
// ============================================================
let DATA = null;
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
  const res = await fetch('data/brackets.json?' + Date.now());
  DATA = await res.json();
  document.getElementById('group-name').textContent = DATA.groupName || '';
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
  const updated = DATA && DATA.lastUpdated ? new Date(DATA.lastUpdated) : null;

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
// Tab 1: Consensus Bracket View
// ============================================================
function renderConsensus() {
  const container = document.getElementById('consensus-content');
  const total = DATA.brackets.length;

  // Build R64 matchup map sorted by displayOrder
  const r64 = DATA.matchups.filter(m => m.round === 1).sort((a, b) => a.displayOrder - b.displayOrder);
  const r64ById = {};
  r64.forEach(m => r64ById[m.id] = m);

  // For each R64 matchup, count picks
  const r64Picks = {};
  r64.forEach(m => {
    let t1 = 0, t2 = 0;
    DATA.brackets.forEach(b => {
      const pick = b.picks.r64.find(p => p.matchupId === m.id);
      if (pick) {
        if (pick.pickedTeamId === m.team1Id) t1++;
        else if (pick.pickedTeamId === m.team2Id) t2++;
      }
    });
    r64Picks[m.id] = { [m.team1Id]: t1, [m.team2Id]: t2 };
  });

  // Build later round slots
  // R32: 16 slots, each from pairs [0,1],[2,3],...,[30,31]
  // S16: 8 slots from quads [0-3],[4-7],...,[28-31]
  // E8:  4 slots from octets [0-7],[8-15],[16-23],[24-31]
  // FF:  2 slots: [0-15], [16-31]
  // Champ: 1 slot

  function buildSlotPicks(roundKey, slotSize) {
    const numSlots = 32 / slotSize;
    const slots = [];
    for (let s = 0; s < numSlots; s++) {
      const minDO = s * slotSize;
      const maxDO = minDO + slotSize - 1;
      // Find all R64 matchup IDs in this range
      const r64IdsInSlot = new Set();
      r64.forEach(m => {
        if (m.displayOrder >= minDO && m.displayOrder <= maxDO) {
          r64IdsInSlot.add(m.id);
        }
      });
      // Count picks for this slot
      const counts = {};
      DATA.brackets.forEach(b => {
        (b.picks[roundKey] || []).forEach(p => {
          if (r64IdsInSlot.has(p.matchupId)) {
            counts[p.pickedTeamId] = (counts[p.pickedTeamId] || 0) + 1;
          }
        });
      });
      slots.push(counts);
    }
    return slots;
  }

  const r32Slots = buildSlotPicks('r32', 2);   // 16 slots
  const s16Slots = buildSlotPicks('sweet16', 4); // 8 slots
  const e8Slots  = buildSlotPicks('elite8', 8);  // 4 slots
  const ffSlots  = buildSlotPicks('finalFour', 16); // 2 slots

  // Championship
  const champCounts = {};
  DATA.brackets.forEach(b => {
    (b.picks.championship || []).forEach(p => {
      champCounts[p.pickedTeamId] = (champCounts[p.pickedTeamId] || 0) + 1;
    });
  });

  // Helper to render a matchup card given team counts object {teamId: count, ...}
  // winnerId is set if decided, team1Id/team2Id define the seeding order
  function renderCard(teamCounts, winnerId) {
    const entries = Object.entries(teamCounts).sort((a, b) => b[1] - a[1]);
    if (entries.length === 0) return '<div class="bk-empty">TBD</div>';

    const totalPicks = entries.reduce((s, e) => s + e[1], 0) || 1;
    const maxCount = entries.length > 0 ? entries[0][1] : 0;
    const decided = !!winnerId;

    let html = `<div class="bk-matchup${decided ? ' decided' : ''}">`;
    // Show top 2 teams (or all if just 1)
    const showTeams = entries.slice(0, 2);
    // If there's a winner, make sure they're in the list
    if (winnerId && !showTeams.find(e => e[0] === winnerId)) {
      const we = entries.find(e => e[0] === winnerId);
      if (we) showTeams[1] = we;
    }

    showTeams.forEach(([tid, count]) => {
      const team = DATA.teams[tid];
      if (!team) return;
      const isWinner = winnerId === tid;
      const isLoser = decided && winnerId !== tid;
      const isFav = !decided && count === maxCount && entries.length > 1;
      let cls = 'bk-team';
      if (isWinner) cls += ' winner';
      if (isLoser) cls += ' loser';
      if (isFav) cls += ' favorite';
      html += `<div class="${cls}">
        <img class="bk-team-logo" src="${team.logo}" alt="${team.abbrev}" loading="lazy">
        <span class="bk-team-seed">${team.seed}</span>
        <span class="bk-team-name">${team.abbrev}</span>
        <span class="bk-team-count">${count}/${total}</span>
      </div>`;
    });

    // If only 1 team, show an empty second row
    if (showTeams.length < 2) {
      html += `<div class="bk-team" style="opacity:0.2;min-height:24px"><span class="bk-team-name" style="color:var(--muted)">---</span></div>`;
    }

    html += '</div>';
    return html;
  }

  // Render R64 matchup card
  function renderR64Card(matchup) {
    const picks = r64Picks[matchup.id];
    return renderCard(picks, matchup.winnerId);
  }

  // Round tab selector
  const rounds = [
    { key: 'r64', label: 'Round of 64', pts: '10 pts', games: 32 },
    { key: 'r32', label: 'Round of 32', pts: '20 pts', games: 16 },
    { key: 'sweet16', label: 'Sweet 16', pts: '40 pts', games: 8 },
    { key: 'elite8', label: 'Elite 8', pts: '80 pts', games: 4 },
    { key: 'finalFour', label: 'Final Four', pts: '160 pts', games: 2 },
    { key: 'championship', label: 'Championship', pts: '320 pts', games: 1 },
  ];

  // Determine the current active round: first round with incomplete games
  const roundMatchups = { r64: 1, r32: 2, sweet16: 3, elite8: 4, finalFour: 5, championship: 6 };
  let defaultRound = 'r64';
  for (const rd of rounds) {
    const roundNum = roundMatchups[rd.key];
    const completed = DATA.matchups.filter(m => m.round === roundNum && m.status === 'COMPLETE').length;
    if (completed >= rd.games) {
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
  openDrawer(d.teamCounts, d.rk, d.winnerId, d.undecidedCounts);
}

function openDrawer(teamCounts, roundKey, winnerId, undecidedCounts) {
  const total = DATA.brackets.length;
  // Sort: winner first if decided, then by count
  const entries = Object.entries(teamCounts).sort((a, b) => {
    if (winnerId) {
      if (a[0] === winnerId) return -1;
      if (b[0] === winnerId) return 1;
    }
    return b[1] - a[1];
  });

  // For each team, find which brackets picked them
  const pickersByTeam = {};
  entries.forEach(([tid]) => { pickersByTeam[tid] = []; });

  const allPickedTeamIds = new Set(entries.map(e => e[0]));
  const neitherPickers = [];

  DATA.brackets.forEach(b => {
    let pickedInThisSlot = false;
    (b.picks[roundKey] || []).forEach(p => {
      if (pickersByTeam[p.pickedTeamId] !== undefined) {
        pickersByTeam[p.pickedTeamId].push({ member: b.member, entry: b.entryName });
        pickedInThisSlot = true;
      }
    });
    if (!pickedInThisSlot && roundKey !== 'r64') {
      neitherPickers.push({ member: b.member, entry: b.entryName });
    }
  });

  let html = '';
  entries.forEach(([tid, count]) => {
    const team = DATA.teams[tid];
    if (!team) return;
    const pickers = pickersByTeam[tid] || [];
    const isWinner = winnerId === tid;
    const isConfirmed = isWinner;
    const countColor = isConfirmed ? 'var(--success)' : 'var(--muted)';
    const nameColor = isConfirmed ? 'var(--success)' : 'var(--white)';
    html += `<div class="drawer-team-group">
      <div class="drawer-team-header">
        <img src="${team.logo}" alt="${team.abbrev}">
        <div class="team-info">
          <div class="team-name" style="color:${nameColor}">(${team.seed}) ${team.name}${isConfirmed ? ' ✓' : ''}</div>
        </div>
        <div class="pick-count" style="color:${countColor}">${count}/${total}</div>
      </div>`;
    pickers.forEach(p => {
      html += `<div class="drawer-picker">
        <span class="picker-name">${p.entry}</span>
      </div>`;
    });
    if (pickers.length === 0) {
      html += `<div style="color:var(--muted);font-size:0.75rem;padding:0.3rem">No one picked this team</div>`;
    }
    html += '</div>';
  });

  // Show undecided feeder teams (TBD opponents) for partial matchups
  if (undecidedCounts && Object.keys(undecidedCounts).length > 0) {
    const undecidedEntries = Object.entries(undecidedCounts).sort((a, b) => b[1] - a[1]);
    undecidedEntries.forEach(([tid, count]) => {
      if (count === 0) return; // skip teams nobody picked
      const team = DATA.teams[tid];
      if (!team) return;
      // Find which brackets picked this undecided team
      const pickers = [];
      DATA.brackets.forEach(b => {
        (b.picks[roundKey] || []).forEach(p => {
          if (p.pickedTeamId === tid) {
            pickers.push({ entry: b.entryName });
          }
        });
      });
      html += `<div class="drawer-team-group">
        <div class="drawer-team-header">
          <img src="${team.logo}" alt="${team.abbrev}">
          <div class="team-info">
            <div class="team-name" style="color:var(--muted)">(${team.seed}) ${team.name} <span style="font-size:0.65rem;font-style:italic;font-weight:400">— TBD</span></div>
          </div>
          <div class="pick-count" style="color:var(--muted)">${count}/${total}</div>
        </div>`;
      pickers.forEach(p => {
        html += `<div class="drawer-picker"><span class="picker-name" style="color:var(--muted)">${p.entry}</span></div>`;
      });
      html += '</div>';
    });
  }

  // Show "neither" section for later rounds
  if (neitherPickers.length > 0) {
    // For partial matchups, recalculate neither — exclude those who picked undecided teams
    let actualNeither = neitherPickers;
    if (undecidedCounts && Object.keys(undecidedCounts).length > 0) {
      const undecidedTeamIds = new Set(Object.keys(undecidedCounts));
      actualNeither = [];
      DATA.brackets.forEach(b => {
        const pickedKnown = (b.picks[roundKey] || []).some(p => allPickedTeamIds.has(p.pickedTeamId));
        const pickedUndecided = (b.picks[roundKey] || []).some(p => undecidedTeamIds.has(p.pickedTeamId));
        if (!pickedKnown && !pickedUndecided) {
          actualNeither.push({ entry: b.entryName });
        }
      });
    }
    if (actualNeither.length > 0) {
      html += `<div class="drawer-team-group">
        <div class="drawer-team-header">
          <div class="team-info">
            <div class="team-name" style="color:var(--muted)">None of these teams</div>
          </div>
          <div class="pick-count" style="color:var(--muted)">${actualNeither.length}/${total}</div>
        </div>`;
      actualNeither.forEach(p => {
        html += `<div class="drawer-picker"><span class="picker-name" style="color:var(--muted)">${p.entry}</span></div>`;
      });
      html += '</div>';
    }
  }

  const hasUndecided = undecidedCounts && Object.keys(undecidedCounts).length > 0;
  let drawerTitle;
  if (entries.length >= 2) {
    drawerTitle = `${DATA.teams[entries[0][0]]?.name || '?'} vs ${DATA.teams[entries[1][0]]?.name || '?'}`;
  } else if (entries.length === 1 && hasUndecided) {
    drawerTitle = `${DATA.teams[entries[0][0]]?.name || '?'} vs TBD`;
  } else if (entries.length === 1) {
    drawerTitle = DATA.teams[entries[0][0]]?.name || 'Detail';
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
  const total = DATA.brackets.length;
  const r64 = DATA.matchups.filter(m => m.round === 1).sort((a, b) => a.displayOrder - b.displayOrder);

  // Build R64 pick counts (needed for R64 view)
  const r64Picks = {};
  r64.forEach(m => {
    let t1 = 0, t2 = 0;
    DATA.brackets.forEach(b => {
      const pick = b.picks.r64.find(p => p.matchupId === m.id);
      if (pick) {
        if (pick.pickedTeamId === m.team1Id) t1++;
        else if (pick.pickedTeamId === m.team2Id) t2++;
      }
    });
    r64Picks[m.id] = { [m.team1Id]: t1, [m.team2Id]: t2 };
  });

  // Build later round slot picks
  function buildSlotPicks(rk, slotSize) {
    const numSlots = 32 / slotSize;
    const slots = [];
    for (let s = 0; s < numSlots; s++) {
      const minDO = s * slotSize;
      const maxDO = minDO + slotSize - 1;
      const r64IdsInSlot = new Set();
      r64.forEach(m => { if (m.displayOrder >= minDO && m.displayOrder <= maxDO) r64IdsInSlot.add(m.id); });
      const counts = {};
      DATA.brackets.forEach(b => {
        (b.picks[rk] || []).forEach(p => {
          if (r64IdsInSlot.has(p.matchupId)) {
            counts[p.pickedTeamId] = (counts[p.pickedTeamId] || 0) + 1;
          }
        });
      });
      slots.push(counts);
    }
    return slots;
  }

  // Store card data for drawer lookups (exposed globally)
  _cardDataStore = [];
  const cardDataStore = _cardDataStore;

  // Render a matchup card (clickable to show picker detail)
  function renderCard(teamCounts, winnerId, rk, gameTime) {
    rk = rk || roundKey;
    const entries = Object.entries(teamCounts).sort((a, b) => b[1] - a[1]);
    if (entries.length === 0) return '<div class="bk-empty">TBD</div>';
    const totalPicks = entries.reduce((s, e) => s + e[1], 0) || 1;
    const maxCount = entries[0][1];
    const decided = !!winnerId;
    const playing = !decided && gameTime && gameTime <= Date.now();
    const cardIdx = cardDataStore.length;
    cardDataStore.push({ teamCounts, rk, winnerId });
    let timeStr = '';
    if (decided) {
      timeStr = 'FINAL';
    } else if (playing) {
      timeStr = 'PLAYING';
    } else if (gameTime) {
      const d = new Date(gameTime);
      timeStr = d.toLocaleString('en-US', { month:'numeric', day:'numeric', timeZone:'America/Chicago' }) + '<br>' + d.toLocaleString('en-US', { hour:'numeric', minute:'2-digit', timeZone:'America/Chicago' });
    }
    let matchupCls = 'bk-matchup';
    if (decided) matchupCls += ' decided';
    else if (playing) matchupCls += ' playing';
    let h = `<div class="${matchupCls}" style="cursor:pointer" onclick="openDrawerByIdx(${cardIdx})">`;
    h += '<div class="bk-matchup-inner">';
    h += '<div class="bk-teams-col">';
    const showTeams = entries.slice(0, 2);
    if (winnerId && !showTeams.find(e => e[0] === winnerId)) {
      const we = entries.find(e => e[0] === winnerId);
      if (we) showTeams[1] = we;
    }
    showTeams.forEach(([tid, count]) => {
      const team = DATA.teams[tid];
      if (!team) return;
      const isWinner = winnerId === tid;
      const isLoser = decided && winnerId !== tid;
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

  if (roundKey === 'r64') {
    // Group by region
    [1, 2, 3, 4].forEach(regionId => {
      const regionMatchups = r64.filter(m => m.region === regionId);
      out += `<div class="region-section"><div class="region-section-title">${REGION_NAMES[regionId]}</div><div class="round-matchups">`;
      regionMatchups.forEach(m => {
        out += renderCard(r64Picks[m.id], m.winnerId, roundKey, m.gameTime);
      });
      out += '</div></div>';
    });
  } else {
    // Build hierarchical round winners — each round's visibility depends on its
    // feeder round's winners, not on R64 results directly.
    const r64Winners = {};
    r64.forEach(m => {
      if (m.winnerId) r64Winners[m.displayOrder] = m.winnerId;
    });

    const laterRounds = [
      { key: 'r32', slotSize: 2, prevKey: 'r64', resultKey: 'r32', regionSlots: [[0,3,'East'],[4,7,'South'],[8,11,'West'],[12,15,'Midwest']] },
      { key: 'sweet16', slotSize: 4, prevKey: 'r32', resultKey: 'sweet16', regionSlots: [[0,1,'East'],[2,3,'South'],[4,5,'West'],[6,7,'Midwest']] },
      { key: 'elite8', slotSize: 8, prevKey: 'sweet16', resultKey: 'elite8', regionSlots: [[0,0,'East'],[1,1,'South'],[2,2,'West'],[3,3,'Midwest']] },
      { key: 'finalFour', slotSize: 16, prevKey: 'elite8', resultKey: 'finalFour', regionSlots: [[0,0,'Semifinal 1 (East/South)'],[1,1,'Semifinal 2 (West/Midwest)']] },
      { key: 'championship', slotSize: 32, prevKey: 'finalFour', resultKey: 'championship', regionSlots: [[0,0,'Championship']] },
    ];

    // Determine winners for each round from pick results (slot index → winnerId).
    // Each round's picks have results indicating whether the team won that round's game:
    // a CORRECT R32 pick means the team won their R32 game.
    const allWinners = { r64: r64Winners };
    const ref = DATA.brackets.length > 0 ? DATA.brackets[0] : null;

    for (const round of laterRounds) {
      const winners = {};
      allWinners[round.key] = winners;
      if (!ref || !round.resultKey) continue;

      // Use the NEXT round's picks to determine this round's winners
      const picks = ref.picks[round.resultKey] || [];
      const numSlots = 32 / round.slotSize;

      for (let s = 0; s < numSlots; s++) {
        const minDO = s * round.slotSize;
        const r64IdsInSlot = new Set();
        r64.forEach(m => {
          if (m.displayOrder >= minDO && m.displayOrder < minDO + round.slotSize) r64IdsInSlot.add(m.id);
        });

        const pick = picks.find(p => round.resultKey === 'championship' || r64IdsInSlot.has(p.matchupId));
        if (!pick || pick.result === 'UNDECIDED') continue;

        if (pick.result === 'CORRECT') {
          winners[s] = pick.pickedTeamId;
        } else {
          // INCORRECT: the other feeder team won
          const prevWinners = allWinners[round.prevKey];
          const team1 = prevWinners[2 * s];
          const team2 = prevWinners[2 * s + 1];
          winners[s] = (pick.pickedTeamId === team1) ? team2 : team1;
        }
      }
    }

    // Render: a matchup is known when both feeders from the previous round have winners
    const roundConfig = laterRounds.find(r => r.key === roundKey);
    if (!roundConfig) { container.innerHTML = out; return; }

    const prevWinners = allWinners[roundConfig.prevKey] || {};
    const numSlots = 32 / roundConfig.slotSize;
    const knownMatchups = [];

    for (let s = 0; s < numSlots; s++) {
      const feeder1 = prevWinners[2 * s];
      const feeder2 = prevWinners[2 * s + 1];

      // Both feeders must be decided to show this matchup
      if (feeder1 === undefined || feeder2 === undefined) continue;

      const actualTeams = new Set([feeder1, feeder2]);

      const minDO = s * roundConfig.slotSize;
      const r64IdsInSlot = new Set();
      r64.forEach(m => {
        if (m.displayOrder >= minDO && m.displayOrder < minDO + roundConfig.slotSize) r64IdsInSlot.add(m.id);
      });

      // Count picks for actual teams in this matchup
      const counts = {};
      actualTeams.forEach(tid => { counts[tid] = 0; });
      const rk = roundKey === 'championship' ? 'championship' : roundKey;
      DATA.brackets.forEach(b => {
        (b.picks[rk] || []).forEach(p => {
          if ((rk === 'championship' || r64IdsInSlot.has(p.matchupId)) && actualTeams.has(p.pickedTeamId)) {
            counts[p.pickedTeamId] = (counts[p.pickedTeamId] || 0) + 1;
          }
        });
      });

      let neitherCount = 0;
      DATA.brackets.forEach(b => {
        const hasPick = (b.picks[rk] || []).some(p =>
          (rk === 'championship' || r64IdsInSlot.has(p.matchupId)) && actualTeams.has(p.pickedTeamId)
        );
        if (!hasPick) neitherCount++;
      });

      // Winner of this matchup (if this round's game has been decided)
      const winner = allWinners[roundKey]?.[s] || null;

      knownMatchups.push({ slot: s, counts, winner, r64IdsInSlot, neitherCount, isPartial: false, undecidedCounts: {} });
    }

    if (knownMatchups.length === 0) {
      out += '<div style="text-align:center;padding:2rem;color:var(--muted)">No matchups decided yet for this round. Check back after more games finish!</div>';
    } else {
      roundConfig.regionSlots.forEach(([start, end, name]) => {
        const regionMatchups = knownMatchups.filter(m => m.slot >= start && m.slot <= end);
        if (regionMatchups.length === 0) return;
        out += `<div class="region-section"><div class="region-section-title">${name}</div><div class="round-matchups">`;
        regionMatchups.forEach(m => {
          out += renderCard(m.counts, m.winner, roundKey);
        });
        out += '</div></div>';
      });
    }
  }

  container.innerHTML = out;
}

// ============================================================
// Tab 2: Bracket Uniqueness
// ============================================================
function getPickVector(bracket) {
  const picks = [];
  ROUND_KEYS.forEach(round => {
    (bracket.picks[round] || []).forEach(pick => picks.push(pick.pickedTeamId));
  });
  return picks;
}

function hammingDistance(v1, v2) {
  let diff = 0;
  const len = Math.min(v1.length, v2.length);
  for (let i = 0; i < len; i++) {
    if (v1[i] !== v2[i]) diff++;
  }
  return diff;
}

function renderUniqueness() {
  const container = document.getElementById('uniqueness-content');
  const brackets = DATA.brackets;
  const vectors = brackets.map(b => getPickVector(b));
  const vecLen = vectors[0] ? vectors[0].length : 1;

  // Pairwise distances
  const distances = [];
  for (let i = 0; i < brackets.length; i++) {
    distances[i] = [];
    for (let j = 0; j < brackets.length; j++) {
      distances[i][j] = i === j ? 0 : hammingDistance(vectors[i], vectors[j]);
    }
  }

  // Originality score = avg distance / vecLen * 100
  const scores = brackets.map((b, i) => {
    const others = distances[i].filter((_, j) => j !== i);
    const avg = others.reduce((s, v) => s + v, 0) / (others.length || 1);
    return { idx: i, bracket: b, score: Math.round(avg / vecLen * 100 * 10) / 10 };
  }).sort((a, b) => b.score - a.score);

  const maxScore = Math.max(...scores.map(s => s.score));
  const minScore = Math.min(...scores.map(s => s.score));

  let html = '<div class="section-title">Originality Leaderboard</div>';
  html += '<div style="overflow-x:auto"><table class="uniqueness-table"><thead><tr>';
  html += '<th>#</th><th>Bracket</th><th>Member</th><th>Score</th><th></th><th></th>';
  html += '</tr></thead><tbody>';

  scores.forEach((s, rank) => {
    const pct = maxScore > 0 ? (s.score / maxScore * 100) : 0;
    const barColor = s.score === maxScore ? 'var(--accent)' : s.score === minScore ? 'var(--muted)' : 'var(--accent)';
    let badge = '';
    if (rank === 0) badge = '<span class="badge badge-unique">Most Unique</span>';
    if (rank === scores.length - 1) badge = '<span class="badge badge-mainstream">Most Mainstream</span>';

    html += `<tr>
      <td class="rank-num">${rank + 1}</td>
      <td style="font-weight:600">${escHtml(s.bracket.entryName)}${badge}</td>
      <td style="color:var(--muted)">${escHtml(s.bracket.member)}</td>
      <td><span class="score-big" style="color:${barColor}">${s.score.toFixed(1)}%</span></td>
      <td><div class="score-bar"><div class="score-bar-fill" style="width:${pct}%;background:${barColor}"></div></div></td>
      <td></td>
    </tr>`;
  });
  html += '</tbody></table></div>';

  // Similarity matrix toggle
  html += '<div class="similarity-toggle"><button onclick="toggleSimilarity()">Show Pairwise Similarity Matrix</button></div>';
  html += '<div class="similarity-wrap" id="simMatrix">';
  html += renderSimilarityMatrix(brackets, vectors, vecLen);
  html += '</div>';

  container.innerHTML = html;
}

function renderSimilarityMatrix(brackets, vectors, vecLen) {
  let html = '<table class="sim-table"><thead><tr><th></th>';
  brackets.forEach(b => { html += `<th>${escHtml(b.entryName).substring(0, 12)}</th>`; });
  html += '</tr></thead><tbody>';

  for (let i = 0; i < brackets.length; i++) {
    html += `<tr><th>${escHtml(brackets[i].entryName).substring(0, 12)}</th>`;
    for (let j = 0; j < brackets.length; j++) {
      if (i === j) {
        html += '<td class="sim-cell-self">-</td>';
      } else {
        const dist = hammingDistance(vectors[i], vectors[j]);
        const similarity = Math.round((1 - dist / vecLen) * 100);
        const hue = similarity > 70 ? '0' : similarity > 40 ? '30' : '140';
        const sat = Math.abs(similarity - 50) * 2;
        html += `<td style="background:hsla(${hue},${sat}%,50%,0.15);color:var(--text)">${similarity}%</td>`;
      }
    }
    html += '</tr>';
  }
  html += '</tbody></table>';
  return html;
}

function toggleSimilarity() {
  document.getElementById('simMatrix').classList.toggle('show');
}

// ============================================================
// Tab 3: What-If Simulator
// ============================================================
// ============================================================
// Tab: Leaderboard
// ============================================================
let lbSortMode = 'current';

function renderLeaderboard() {
  const container = document.getElementById('leaderboard-content');
  let html = '<div class="lb-pills">';
  html += `<button class="lb-pill${lbSortMode === 'current' ? ' active' : ''}" onclick="setLbSort('current')">Current Points</button>`;
  html += `<button class="lb-pill${lbSortMode === 'possible' ? ' active' : ''}" onclick="setLbSort('possible')">Possible Points</button>`;
  html += `<button class="lb-pill${lbSortMode === 'correct' ? ' active' : ''}" onclick="setLbSort('correct')">Most Correct</button>`;
  html += '</div>';

  function countPicks(bracket) {
    const rounds = ['r64', 'r32', 's16', 'e8', 'ff', 'championship'];
    let correct = 0, total = 0;
    rounds.forEach(r => {
      (bracket.picks[r] || []).forEach(p => {
        if (p.result === 'CORRECT') { correct++; total++; }
        else if (p.result === 'INCORRECT') { total++; }
      });
    });
    return { correct, total };
  }

  const eliminatedTeams = new Set();
  DATA.matchups.forEach(m => {
    if (m.winnerId) {
      if (m.team1Id && m.team1Id !== m.winnerId) eliminatedTeams.add(m.team1Id);
      if (m.team2Id && m.team2Id !== m.winnerId) eliminatedTeams.add(m.team2Id);
    }
  });

  const sorted = [...DATA.brackets].sort((a, b) => {
    if (lbSortMode === 'current') return b.score - a.score || b.maxPossible - a.maxPossible;
    if (lbSortMode === 'correct') {
      const ac = countPicks(a), bc = countPicks(b);
      return bc.correct - ac.correct || b.score - a.score;
    }
    return b.maxPossible - a.maxPossible || b.score - a.score;
  });

  html += '<div class="lb-list">';
  function ordinal(n) {
    const s = ['th','st','nd','rd'];
    const v = n % 100;
    return n + (s[(v - 20) % 10] || s[v] || s[0]);
  }

  sorted.forEach((b, i) => {
    const picks = countPicks(b);
    const pts = lbSortMode === 'current' ? b.score : lbSortMode === 'possible' ? b.maxPossible : picks.correct;
    const subPts = lbSortMode === 'current' ? `/${b.maxPossible}` : lbSortMode === 'possible' ? `/${b.score}` : `/${picks.total}`;
    // Handle ties — same rank if same primary value
    const getPrimary = x => lbSortMode === 'current' ? x.score : lbSortMode === 'possible' ? x.maxPossible : countPicks(x).correct;
    const rank = (i === 0 || pts !== getPrimary(sorted[i-1]))
      ? ordinal(i + 1) : ordinal(sorted.findIndex(x => getPrimary(x) === pts) + 1);
    const rankNum = sorted.findIndex(x => getPrimary(x) === pts) + 1;
    const placeClass = rankNum === 1 ? ' lb-1st' : rankNum === 2 ? ' lb-2nd' : rankNum === 3 ? ' lb-3rd' : '';
    const champTeam = DATA.teams[b.champion];
    const champOut = eliminatedTeams.has(b.champion);
    const champLogo = champTeam ? `<img src="${champTeam.logo}" alt="${champTeam.abbrev}" loading="lazy" style="width:32px;height:32px;${champOut ? 'filter:grayscale(100%);opacity:0.5;' : ''}">` : '';
    const bIdx = DATA.brackets.indexOf(b);
    html += `<div class="lb-row${placeClass}" style="cursor:pointer" onclick="openBracketModal(${bIdx})">
      <div class="lb-rank">${rank}</div>
      ${champLogo ? `<div class="lb-champ">${champLogo}</div>` : ''}
      <div class="lb-info">
        <div class="lb-entry-name">${b.entryName}</div>
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
  const b = DATA.brackets[idx];
  if (!b) return;
  const total = DATA.brackets.length;
  const champTeam = DATA.teams[b.champion];
  const name = ENTRY_NAMES[b.entryName] || b.member;
  const pct = Math.round(b.percentile * 100);

  // Final Four teams
  // Sort Final Four by region: East(1) top-left, West(3) top-right, South(2) bottom-left, Midwest(4) bottom-right
  const ffOrder = [1, 3, 2, 4]; // grid order: top-left, top-right, bottom-left, bottom-right
  // Build set of teams eliminated from the tournament
  const eliminatedTeams = new Set();
  DATA.matchups.forEach(m => {
    if (m.winnerId) {
      if (m.team1Id && m.team1Id !== m.winnerId) eliminatedTeams.add(m.team1Id);
      if (m.team2Id && m.team2Id !== m.winnerId) eliminatedTeams.add(m.team2Id);
    }
  });

  const ffSorted = ffOrder.map(region => {
    const tid = b.finalFour.find(id => DATA.teams[id] && DATA.teams[id].region === region);
    return tid ? { ...DATA.teams[tid], id: tid } : null;
  });
  // Teams that actually made the Final Four (won E8 = appear in round 5 matchups, or won a round 4 matchup)
  const actualFF = new Set();
  DATA.matchups.forEach(m => {
    if (m.round === 4 && m.winnerId) actualFF.add(m.winnerId);
    if (m.round === 5) { if (m.team1Id) actualFF.add(m.team1Id); if (m.team2Id) actualFF.add(m.team2Id); }
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

  // Pick record (all rounds)
  let wins = 0, losses = 0;
  ['r64', 'r32', 's16', 'e8', 'ff', 'championship'].forEach(r => {
    (b.picks[r] || []).forEach(p => {
      if (p.result === 'CORRECT') wins++;
      else if (p.result === 'INCORRECT') losses++;
    });
  });
  const pickPct = (wins + losses) > 0 ? Math.round(wins / (wins + losses) * 100) : 0;

  let html = `<div style="font-weight:700;font-size:1.1rem;color:var(--white)">${b.entryName}</div>`;
  html += `<div style="font-size:0.7rem;color:var(--muted);margin-bottom:1rem">${name}</div>`;

  // Champion
  const champEliminated = eliminatedTeams.has(b.champion);
  if (champTeam) {
    html += `<div class="bracket-modal-champ">
      <img src="${champTeam.logo}" alt="${champTeam.abbrev}" loading="lazy" style="${champEliminated ? 'filter:grayscale(100%);opacity:0.5;' : ''}">
      <div class="bracket-modal-champ-info">
        <div class="bracket-modal-champ-name">${champTeam.name}</div>
        <div class="bracket-modal-champ-sub">${champTeam.seed} seed · ${champTeam.record} · Champion Pick</div>
      </div>
    </div>`;
  }

  // Bracket status badge
  if (b.eliminated) {
    html += `<div class="bracket-modal-eliminated" style="background:rgba(239,68,68,0.1);color:var(--danger)">Eliminated</div>`;
  } else if (losses > 0) {
    html += `<div class="bracket-modal-eliminated" style="background:rgba(239,68,68,0.1);color:var(--danger)">Perfect Bracket Busted</div>`;
  } else if (wins > 0) {
    html += `<div class="bracket-modal-eliminated" style="background:rgba(34,197,94,0.1);color:#22c55e">Perfect Bracket</div>`;
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
  html += `<div class="bracket-modal-stat">
    <div class="bracket-modal-stat-value">${wins}/${wins + losses}</div>
    <div class="bracket-modal-stat-label">Pick Record</div>
  </div>`;
  html += `<div class="bracket-modal-stat">
    <div class="bracket-modal-stat-value">${pickPct}%</div>
    <div class="bracket-modal-stat-label">Pick Accuracy</div>
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
// Tab: What-If Simulator
// ============================================================
function initWhatIf() {
  // Deep clone
  whatIfState = {
    matchups: JSON.parse(JSON.stringify(DATA.matchups)),
    brackets: JSON.parse(JSON.stringify(DATA.brackets)),
    pointsPerRound: DATA.pointsPerRound,
    originalScores: DATA.brackets.map(b => b.score),
    originalMaxPossible: DATA.brackets.map(b => b.maxPossible)
  };
  renderWhatIf();
}

function applyWhatIf(matchupId, winnerTeamId) {
  const matchup = whatIfState.matchups.find(m => m.id === matchupId);
  if (!matchup) return;
  const loserId = matchup.team1Id === winnerTeamId ? matchup.team2Id : matchup.team1Id;
  matchup.winnerId = winnerTeamId;

  whatIfState.brackets.forEach(bracket => {
    let score = 0, maxPossible = 0;
    ROUND_KEYS.forEach((roundKey, idx) => {
      const pts = whatIfState.pointsPerRound[idx];
      (bracket.picks[roundKey] || []).forEach(pick => {
        // Eliminate ALL picks for the losing team across all rounds
        if (pick.pickedTeamId === loserId && pick.result === 'UNDECIDED') {
          pick.result = 'ELIMINATED';
        }
        // Mark correct for this specific matchup
        if (pick.matchupId === matchupId && pick.pickedTeamId === winnerTeamId && pick.result !== 'CORRECT') {
          pick.result = 'CORRECT';
        }
        // Mark incorrect for this specific matchup
        if (pick.matchupId === matchupId && pick.pickedTeamId === loserId) {
          pick.result = 'INCORRECT';
        }
        // Tally
        if (pick.result === 'CORRECT') score += pts;
        if (pick.result === 'CORRECT' || pick.result === 'UNDECIDED') maxPossible += pts;
      });
    });
    bracket.score = score;
    bracket.maxPossible = maxPossible;
  });
  renderWhatIf();
}

function resetWhatIf() {
  initWhatIf();
}

function renderWhatIf() {
  const container = document.getElementById('whatif-content');
  let html = '<div class="whatif-layout">';

  // Left: bracket matchups
  html += '<div>';
  html += '<div class="section-title">Select Winners</div>';
  html += '<div class="whatif-bracket">';

  const matchups = whatIfState.matchups.filter(m => m.round === 1);
  const byRegion = {};
  matchups.forEach(m => {
    if (!byRegion[m.region]) byRegion[m.region] = [];
    byRegion[m.region].push(m);
  });

  [1,2,3,4].forEach(regionId => {
    const regionMatchups = (byRegion[regionId] || []).sort((a, b) => a.displayOrder - b.displayOrder);
    html += `<div class="whatif-region"><div class="region-header">${REGION_NAMES[regionId]}</div>`;
    regionMatchups.forEach(m => {
      const team1 = DATA.teams[m.team1Id];
      const team2 = DATA.teams[m.team2Id];
      if (!team1 || !team2) return;
      const decided = !!m.winnerId;
      const t1Won = m.winnerId === m.team1Id;
      const t2Won = m.winnerId === m.team2Id;

      let t1Class = '';
      let t2Class = '';
      if (decided) {
        t1Class = t1Won ? 'picked-correct' : 'picked-incorrect';
        t2Class = t2Won ? 'picked-correct' : 'picked-incorrect';
      }

      const clickable = !decided;
      html += '<div class="whatif-matchup">';
      html += `<div class="whatif-team ${t1Class}" ${clickable ? `onclick="applyWhatIf('${m.id}','${m.team1Id}')"` : ''}>
        <img class="team-logo" src="${team1.logo}" alt="${team1.abbrev}" loading="lazy">
        <span class="team-seed">${team1.seed}</span>
        <span class="team-name">${team1.name}</span>
      </div>`;
      html += '<div class="whatif-divider"></div>';
      html += `<div class="whatif-team ${t2Class}" ${clickable ? `onclick="applyWhatIf('${m.id}','${m.team2Id}')"` : ''}>
        <img class="team-logo" src="${team2.logo}" alt="${team2.abbrev}" loading="lazy">
        <span class="team-seed">${team2.seed}</span>
        <span class="team-name">${team2.name}</span>
      </div>`;
      html += '</div>';
    });
    html += '</div>';
  });
  html += '</div></div>';

  // Right: leaderboard
  html += '<div class="leaderboard-panel">';
  html += '<div class="leaderboard-header"><h3>Leaderboard</h3><button class="btn-reset" onclick="resetWhatIf()">Reset</button></div>';

  const sorted = whatIfState.brackets.map((b, i) => ({
    bracket: b,
    idx: i,
    delta: b.score - whatIfState.originalScores[i]
  })).sort((a, b) => b.bracket.score - a.bracket.score || b.bracket.maxPossible - a.bracket.maxPossible);

  html += '<table class="lb-table"><thead><tr><th>#</th><th>Bracket</th><th>Pts</th><th>Max</th><th>+/-</th></tr></thead><tbody>';
  sorted.forEach((s, rank) => {
    const deltaClass = s.delta > 0 ? 'delta-pos' : s.delta < 0 ? 'delta-neg' : 'delta-zero';
    const deltaStr = s.delta > 0 ? '+' + s.delta : s.delta === 0 ? '-' : '' + s.delta;
    html += `<tr>
      <td class="rank-num" style="font-size:0.85rem">${rank + 1}</td>
      <td><span class="lb-entry-name" title="${escHtml(s.bracket.entryName)}">${escHtml(s.bracket.entryName)}</span></td>
      <td><span class="lb-score">${s.bracket.score}</span></td>
      <td><span class="lb-max">${s.bracket.maxPossible}</span></td>
      <td><span class="lb-delta ${deltaClass}">${deltaStr}</span></td>
    </tr>`;
  });
  html += '</tbody></table></div>';

  html += '</div>';
  container.innerHTML = html;
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
