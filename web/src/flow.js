import { $, S, api, apiPostJson } from './state.js';
import { openFile } from './tabs.js';
import { renderMermaidBlocks } from './mermaid.js';

const modal = $('#flow-modal');
const graph = $('#flow-graph');
const message = $('#flow-message');
const scopeSelect = $('#flow-scope');
const threadSelect = $('#flow-thread');
let flowSeq = 0;
let planRunning = false;

const statusNames = new Set(['added', 'modified', 'deleted', 'context', 'planned', 'unplanned']);
const layerNames = { ui: 'UI', api: 'API', service: 'Service', db: 'Database' };
const nodeClasses = new Set(['added', 'modified', 'deleted', 'context', 'planned', 'unplanned']);
const edgeKinds = new Set(['call', 'fetch', 'query']);

function nodeLabel(value) {
  return String(value || 'symbol').replace(/["\\\[\]{}<>|;]/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 150) || 'symbol';
}

function mermaidSource(nodes, edges) {
  const ids = new Map();
  const classes = new Map();
  const groups = ['ui', 'api', 'service', 'db'];
  let n = 0;
  for (const node of nodes) ids.set(node.id, `flowNode${n++}`);
  let source = 'flowchart LR\n';
  for (const layer of groups) {
    const members = nodes.filter(node => node.layer === layer);
    if (!members.length) continue;
    source += `  subgraph ${layer}Group["${layerNames[layer]}"]\n`;
    for (const node of members) {
      const id = ids.get(node.id);
      const status = statusNames.has(node.status) ? node.status : 'modified';
      const loc = node.path + (node.line > 0 ? `:${node.line}` : '');
      source += `    ${id}["${nodeLabel(node.label)} · ${nodeLabel(loc)}"]\n`;
      const displayStatus = node.unplanned && node.status !== 'context' ? 'unplanned' : status;
      const cls = nodeClasses.has(displayStatus) ? displayStatus : 'modified';
      classes.set(cls, [...(classes.get(cls) || []), id]);
    }
    source += '  end\n';
  }
  for (const edge of edges) {
    const from = ids.get(edge.from), to = ids.get(edge.to);
    if (!from || !to) continue;
    const kind = edgeKinds.has(edge.kind) ? edge.kind : 'call';
    source += `  ${from} -->|${kind}| ${to}\n`;
  }
  if (classes.size) source += `  classDef added fill:#18382a,stroke:#54bf88,color:#e3f5e8\n  classDef modified fill:#282137,stroke:#b68be9,color:#eee4ff\n  classDef deleted fill:#3a2024,stroke:#e47777,color:#ffe5e5\n  classDef context fill:#20252c,stroke:#68717d,color:#d5dae0\n  classDef planned fill:#251d33,stroke:#c5a2ff,color:#f1e7ff,stroke-dasharray:5 4\n  classDef unplanned fill:#3c2b19,stroke:#e2a54f,color:#fff0d7\n`;
  for (const [cls, members] of classes) source += `  class ${members.join(',')} ${cls}\n`;
  return { source, ids };
}

function planFromThread(thread) {
  const turns = thread?.turns || [];
  for (let i = turns.length - 1; i >= 0; i--) {
    const reply = turns[i]?.reply || '';
    const blocks = [...reply.matchAll(/```flowjson\s*([\s\S]*?)```/gi)];
    if (!blocks.length) continue;
    try {
      const parsed = JSON.parse(blocks[blocks.length - 1][1]);
      if (Array.isArray(parsed.nodes) && Array.isArray(parsed.edges)) return parsed;
    } catch { /* Ignore malformed agent output and try an earlier turn. */ }
  }
  return null;
}

function overlayPlan(actual, plan) {
  if (!plan) return actual;
  const actualKeys = new Map(actual.nodes.map(node => [`${node.path}#${node.label}`, node]));
  const plannedKeys = new Set(plan.nodes.filter(node => node && typeof node.path === 'string' && typeof node.label === 'string').map(node => `${node.path}#${node.label}`));
  for (const node of actual.nodes) {
    if (node.status !== 'context') node.unplanned = !plannedKeys.has(`${node.path}#${node.label}`);
  }
  const planNodes = [];
  const planIDs = new Map();
  for (const node of plan.nodes) {
    if (!node || typeof node.id !== 'string' || typeof node.path !== 'string' || typeof node.label !== 'string') continue;
    const match = actualKeys.get(`${node.path}#${node.label}`);
    if (match) {
      match.planned = true;
      planIDs.set(node.id, match.id);
      continue;
    }
    const plannedID = `planned:${node.id}`;
    planIDs.set(node.id, plannedID);
    planNodes.push({
      id: plannedID,
      label: node.label,
      layer: layerNames[node.layer?.toLowerCase()] ? node.layer.toLowerCase() : 'service',
      path: node.path,
      line: Number.isFinite(Number(node.line)) ? Math.max(0, Number(node.line)) : 0,
      status: 'planned',
    });
  }
  const planEdges = [];
  const seen = new Set(actual.edges.map(edge => `${edge.from}\0${edge.to}\0${edge.kind}`));
  for (const edge of plan.edges || []) {
    if (!edge) continue;
    const from = planIDs.get(edge.from);
    const to = planIDs.get(edge.to);
    if (!from || !to || from === to) continue;
    const kind = edgeKinds.has(edge.kind) ? edge.kind : 'call';
    const key = `${from}\0${to}\0${kind}`;
    if (seen.has(key)) continue;
    seen.add(key);
    planEdges.push({ from, to, kind });
  }
  return { ...actual, nodes: [...actual.nodes, ...planNodes], edges: [...actual.edges, ...planEdges] };
}

function showMessage(text) {
  message.textContent = text;
  message.hidden = !text;
}

function renderFlow(data) {
  graph.replaceChildren();
  showMessage('');
  if (!data.nodes?.length) {
    showMessage(scopeSelect.value === 'thread' ? 'No changed files were recorded for this thread turn.' : 'No code changes in this scope yet.');
    $('#flow-summary').textContent = 'No changed symbols found';
    return;
  }
  const pre = document.createElement('div');
  pre.className = 'md-pre';
  pre.dataset.lang = 'mermaid';
  const source = document.createElement('pre');
  const code = document.createElement('code');
  code.textContent = mermaidSource(data.nodes, data.edges || []).source;
  source.append(code);
  pre.append(source);
  graph.append(pre);
  const changed = data.nodes.filter(node => node.status !== 'context');
  $('#flow-summary').textContent = `${changed.length} changed ${changed.length === 1 ? 'symbol' : 'symbols'} · ${data.edges.length} relationships${data.scope === 'thread' ? ' · this thread' : ''}`;
  const renderID = flowSeq;
  renderMermaidBlocks(graph, () => renderID === flowSeq && !modal.hidden).then(() => {
    if (renderID !== flowSeq) return;
    const svg = graph.querySelector('.md-mermaid-canvas svg');
    if (!svg) return;
    const { ids } = mermaidSource(data.nodes, data.edges || []);
    for (const node of data.nodes) {
      const mermaidID = ids.get(node.id);
      if (!mermaidID) continue;
      const element = svg.querySelector(`[id^="flowchart-${mermaidID}-"]`);
      if (!element) continue;
      element.classList.add('flow-node-clickable');
      element.setAttribute('tabindex', '0');
      element.setAttribute('role', 'button');
      element.setAttribute('aria-label', `${node.label}, ${node.path}${node.line ? ` line ${node.line}` : ''}, ${node.status}`);
      element.addEventListener('click', async () => {
        modal.hidden = true;
        try { await openFile(node.path, { line: node.line || undefined, view: 'diff' }); }
        catch { await openFile(node.path, { line: node.line || undefined }); }
      });
      element.addEventListener('keydown', event => {
        if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); element.dispatchEvent(new MouseEvent('click', { bubbles: true })); }
      });
    }
  });
}

async function loadThreadOptions() {
  try {
    const data = await api('/api/threads');
    const threads = data.threads || [];
    const previous = threadSelect.value;
    threadSelect.replaceChildren(...threads.map(thread => {
      const option = document.createElement('option');
      option.value = thread.id;
      option.textContent = thread.title + (thread.running ? ' · working' : '');
      return option;
    }));
    if (threads.some(thread => thread.id === previous)) threadSelect.value = previous;
    threadSelect.hidden = !threads.length;
    $('#flow-plan').disabled = !threads.length || planRunning;
    return threads;
  } catch {
    threadSelect.replaceChildren();
    threadSelect.hidden = true;
    $('#flow-plan').disabled = true;
    return [];
  }
}

export async function refreshFlow() {
  if (modal.hidden) return;
  const current = ++flowSeq;
  const scope = scopeSelect.value;
  graph.replaceChildren();
  showMessage('Indexing changed symbols and relationships…');
  try {
    if (scope === 'thread') {
      await loadThreadOptions();
      if (!threadSelect.value) {
        showMessage('No coding threads are available in this workspace.');
        $('#flow-summary').textContent = 'No thread selected';
        return;
      }
    }
    if (current !== flowSeq) return;
    const threadID = scope === 'thread' ? threadSelect.value : '';
    const data = await api('/api/flow', { scope, id: threadID });
    if (current !== flowSeq) return;
    let result = data;
    if (scope === 'thread' && threadID) {
      const thread = await api('/api/threads/get', { id: threadID });
      if (current !== flowSeq) return;
      result = overlayPlan(data, planFromThread(thread));
    }
    renderFlow(result);
  } catch (error) {
    if (current !== flowSeq) return;
    graph.replaceChildren();
    showMessage(error.message || 'Could not build the change flow.');
  }
}

export function showFlow() {
  modal.hidden = false;
  document.body.classList.add('flow-active');
  scopeSelect.querySelector('option[value="pr"]').hidden = !Boolean(S.meta?.pr);
  if (scopeSelect.value === 'pr' && !S.meta?.pr) scopeSelect.value = 'working';
  const thread = scopeSelect.value === 'thread';
  threadSelect.hidden = !thread;
  $('#flow-plan').hidden = !thread;
  refreshFlow();
}

export function initFlow() {
  $('#btn-flow-view')?.addEventListener('click', showFlow);
  $('#flow-close')?.addEventListener('click', () => { modal.hidden = true; document.body.classList.remove('flow-active'); });
  $('#flow-refresh')?.addEventListener('click', refreshFlow);
  scopeSelect.addEventListener('change', async () => {
    const thread = scopeSelect.value === 'thread';
    threadSelect.hidden = !thread;
    $('#flow-plan').hidden = !thread;
    if (thread) await loadThreadOptions();
    refreshFlow();
  });
  threadSelect.addEventListener('change', refreshFlow);
  modal.addEventListener('click', event => { if (event.target === modal) { modal.hidden = true; document.body.classList.remove('flow-active'); } });
  $('#flow-plan')?.addEventListener('click', planFlow);
  addEventListener('keydown', event => {
    if (event.key === 'Escape' && !modal.hidden) {
      event.stopImmediatePropagation();
      modal.hidden = true;
      document.body.classList.remove('flow-active');
    }
  });
  $('#flow-plan').hidden = scopeSelect.value !== 'thread';
  if (S.meta?.pr) scopeSelect.querySelector('option[value="pr"]').hidden = false;
}

async function planFlow() {
  const id = threadSelect.value;
  if (!id || planRunning) return;
  const prompt = 'Do not edit or create files. Analyze the latest user task in this thread and reply with one fenced `flowjson` block only, using this JSON schema: {"nodes":[{"id":"stable-id","label":"symbol or file","layer":"ui|api|service|db","path":"workspace/relative/path","line":1,"status":"planned"}],"edges":[{"from":"stable-id","to":"stable-id","kind":"call|fetch|query"}]}. Include only the likely code path from UI through backend to data storage. Keep node IDs stable and edges between IDs. Do not include prose outside the JSON fence.';
  planRunning = true;
  $('#flow-plan').disabled = true;
  $('#flow-plan').textContent = 'Planning…';
  showMessage('Asking the selected thread harness for a read-only flow plan…');
  try {
    await apiPostJson('/api/threads/send', { id, message: prompt });
    const maxPolls = 120;
    for (let i = 0; i < maxPolls; i++) {
      await new Promise(resolve => setTimeout(resolve, 1000));
      if (modal.hidden || scopeSelect.value !== 'thread' || threadSelect.value !== id) break;
      const thread = await api('/api/threads/get', { id });
      const last = thread.turns?.[thread.turns.length - 1];
      if (!last?.running) {
        if (!planFromThread(thread)) showMessage('The harness finished without a valid flowjson block. Open the thread to inspect its reply.');
        await refreshFlow();
        break;
      }
    }
  } catch (error) {
    showMessage(error.message || 'Could not start the flow plan.');
  } finally {
    planRunning = false;
    $('#flow-plan').textContent = 'Plan flow';
    $('#flow-plan').disabled = !threadSelect.value;
  }
}

export function refreshFlowIfOpen() {
  if (!modal.hidden) refreshFlow();
}

