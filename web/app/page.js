'use client';

import { useEffect, useState, useCallback, useMemo, useRef } from 'react';

function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function getFileExtension(name) {
  if (!name) return '(no ext)';
  const base = name.split('/').pop() || name;
  const idx = base.lastIndexOf('.');
  if (idx <= 0 || idx === base.length - 1) return '(no ext)';
  return base.slice(idx).toLowerCase();
}

const EXTENSION_PALETTE = {
  '.go': 202,
  '.js': 46,
  '.mjs': 42,
  '.ts': 212,
  '.tsx': 218,
  '.css': 268,
  '.md': 152,
  '.json': 28,
  '.yml': 14,
  '.yaml': 14,
  '.sql': 332,
  '.sh': 125,
  '.mod': 185,
  '.sum': 190,
  '.jpg': 350,
  '.png': 340,
  '.svg': 290,
  '(no ext)': 215,
};

function getExtensionStyle(ext) {
  let hue = EXTENSION_PALETTE[ext];
  if (hue === undefined) {
    let hash = 0;
    for (let i = 0; i < ext.length; i++) {
      hash = (hash * 31 + ext.charCodeAt(i)) % 360;
    }
    hue = Math.abs(hash);
  }
  return {
    swatch: `hsl(${hue}, 72%, 48%)`,
    cushion: `radial-gradient(circle at 30% 26%, hsl(${hue}, 82%, 62%) 0%, hsl(${hue}, 74%, 43%) 58%, hsl(${hue}, 78%, 24%) 100%)`,
  };
}

function getFileCategory(ext) {
  switch (ext) {
    case '.go':
    case '.js':
    case '.mjs':
    case '.ts':
    case '.tsx':
    case '.jsx':
    case '.css':
    case '.html':
    case '.sql':
    case '.sh':
    case '.py':
    case '.rs':
      return 'Source Code';
    case '.mp4':
    case '.mkv':
    case '.mov':
    case '.webm':
    case '.flac':
    case '.mp3':
    case '.wav':
      return 'Video & Audio';
    case '.jpg':
    case '.jpeg':
    case '.png':
    case '.svg':
    case '.gif':
    case '.webp':
      return 'Images & Vector';
    case '.bin':
    case '.parquet':
    case '.wasm':
    case '.pt':
    case '.onnx':
      return 'AI & Datasets';
    case '.gz':
    case '.tar':
    case '.zip':
    case '.7z':
      return 'Archives';
    case '.md':
    case '.json':
    case '.yaml':
    case '.yml':
    case '.mod':
    case '.sum':
    case '.txt':
      return 'Config & Docs';
    case '.log':
      return 'Logs';
    default:
      return 'Binaries & Other';
  }
}

function buildWinDirTree(computer, scanDetailsMap) {
  if (!computer) return null;
  const rootNode = {
    key: `comp:${computer.id}`,
    name: computer.hostname || computer.id,
    kind: 'computer',
    isDir: true,
    size: 0,
    subtreeBytes: 0,
    fileCount: 0,
    dirCount: 0,
    hash: '',
    driveId: (computer.drives || []).join(', '),
    childrenMap: new Map(),
    children: [],
    leafFiles: [],
  };

  const directories = computer.directories || [];
  for (const dirSummary of directories) {
    const detail = scanDetailsMap[dirSummary.id];
    const dirKey = `scan:${dirSummary.id}`;
    const scanNode = {
      key: dirKey,
      scanId: dirSummary.id,
      name: dirSummary.root_path || dirSummary.id,
      kind: 'root',
      isDir: true,
      size: 0,
      subtreeBytes: 0,
      fileCount: 0,
      dirCount: 0,
      hash: dirSummary.root_hash || '',
      driveId: dirSummary.drive_id || 'default-drive',
      version: dirSummary.version || 1,
      childrenMap: new Map(),
      children: [],
    };
    rootNode.childrenMap.set(dirKey, scanNode);

    const entries = detail?.entries || [];
    if (entries.length === 0 && (dirSummary.total_bytes || 0) > 0 && !detail) {
      // Summary placeholder while detail is still loading
      scanNode.subtreeBytes = dirSummary.total_bytes || 0;
      scanNode.fileCount = dirSummary.total_files || 0;
      scanNode.dirCount = dirSummary.total_dirs || 0;
      continue;
    }

    for (const entry of entries) {
      if (!entry.rel_path || entry.rel_path === '.') {
        if (entry.hash && !scanNode.hash) {
          scanNode.hash = entry.hash;
        }
        continue;
      }

      const parts = entry.rel_path.split('/').filter(Boolean);
      let current = scanNode;
      let pathSoFar = '';

      for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        pathSoFar = pathSoFar ? `${pathSoFar}/${part}` : part;
        const isLast = i === parts.length - 1;
        const childKey = `${dirKey}:${pathSoFar}`;

        let child = current.childrenMap.get(childKey);
        if (!child) {
          const isDirNode = isLast ? Boolean(entry.is_dir) : true;
          child = {
            key: childKey,
            scanId: dirSummary.id,
            name: part,
            relPath: pathSoFar,
            absPath: entry.abs_path || `${dirSummary.root_path}/${pathSoFar}`,
            kind: isDirNode ? 'dir' : 'file',
            isDir: isDirNode,
            size: !isDirNode && isLast ? entry.size || 0 : 0,
            subtreeBytes: 0,
            fileCount: 0,
            dirCount: 0,
            hash: isLast ? entry.hash || '' : '',
            owner: isLast ? entry.owner || '' : '',
            mode: isLast ? entry.mode || '' : '',
            ext: !isDirNode ? getFileExtension(part) : '',
            childrenMap: new Map(),
            children: [],
          };
          current.childrenMap.set(childKey, child);
        } else if (isLast) {
          child.isDir = Boolean(entry.is_dir);
          child.kind = child.isDir ? 'dir' : 'file';
          if (!child.isDir) {
            child.size = entry.size || 0;
            child.ext = getFileExtension(part);
          }
          if (entry.hash) child.hash = entry.hash;
          if (entry.abs_path) child.absPath = entry.abs_path;
          if (entry.owner) child.owner = entry.owner;
          if (entry.mode) child.mode = entry.mode;
        }
        current = child;
      }
    }
  }

  function finalizeNode(node) {
    const childList = Array.from(node.childrenMap.values());
    if (childList.length === 0) {
      if (!node.isDir) {
        node.subtreeBytes = node.size || 0;
        node.fileCount = 1;
        node.dirCount = 0;
      }
      node.children = [];
      delete node.childrenMap;
      return;
    }

    let bytes = 0;
    let files = 0;
    let dirs = 0;

    for (const ch of childList) {
      ch.parentKey = node.key;
      finalizeNode(ch);
      bytes += ch.subtreeBytes || 0;
      if (ch.isDir) {
        dirs += 1 + (ch.dirCount || 0);
        files += ch.fileCount || 0;
      } else {
        files += 1;
      }
    }

    childList.sort((a, b) => {
      if (b.subtreeBytes !== a.subtreeBytes) return b.subtreeBytes - a.subtreeBytes;
      if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
      return a.name.localeCompare(b.name);
    });

    for (const ch of childList) {
      ch.pctOfParent = bytes > 0 ? (ch.subtreeBytes / bytes) * 100 : 0;
    }

    node.subtreeBytes = bytes;
    node.fileCount = files;
    node.dirCount = dirs;
    node.children = childList;
    delete node.childrenMap;
  }

  finalizeNode(rootNode);
  rootNode.pctOfParent = 100;

  const totalRootBytes = rootNode.subtreeBytes || 0;
  function assignRootPct(node) {
    node.pctOfRoot = totalRootBytes > 0 ? (node.subtreeBytes / totalRootBytes) * 100 : 0;
    for (const ch of node.children || []) {
      assignRootPct(ch);
    }
  }
  assignRootPct(rootNode);

  return rootNode;
}

function findNodeByKey(node, key) {
  if (!node) return null;
  if (node.key === key) return node;
  for (const ch of node.children || []) {
    const found = findNodeByKey(ch, key);
    if (found) return found;
  }
  return null;
}

function getNodeAncestors(rootNode, targetKey) {
  if (!rootNode || !targetKey) return [];
  const trail = [];
  let cursor = findNodeByKey(rootNode, targetKey);
  while (cursor) {
    trail.unshift(cursor);
    if (!cursor.parentKey) break;
    cursor = findNodeByKey(rootNode, cursor.parentKey);
  }
  return trail;
}

function collectLeafFiles(node, out = []) {
  if (!node) return out;
  if (!node.isDir) {
    out.push(node);
    return out;
  }
  for (const ch of node.children || []) {
    collectLeafFiles(ch, out);
  }
  return out;
}

const HEXBIN_PERIOD_AXIS = {
  day: ['00h', '02h', '04h', '06h', '08h', '10h', '12h', '14h', '16h', '18h', '20h', '22h', '23h', 'Now'],
  week: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'],
  month: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec', 'Jan', 'Feb'],
  year: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec', 'Jan', 'Feb'],
};

const HEXBIN_OP_PALETTES = {
  create: {
    key: 'create',
    label: 'CREATE',
    pillBg: '#059669',
    pillText: '#ffffff',
    levels: {
      6: { fill: '#ecfdf5', stroke: '#a7f3d0', text: '#047857' },
      25: { fill: '#d1fae5', stroke: '#6ee7b7', text: '#065f46' },
      42: { fill: '#a7f3d0', stroke: '#34d399', text: '#064e3b' },
      61: { fill: '#6ee7b7', stroke: '#10b981', text: '#064e3b' },
      86: { fill: '#34d399', stroke: '#059669', text: '#022c22' },
      99: { fill: '#10b981', stroke: '#047857', text: '#ffffff' },
      100: { fill: '#047857', stroke: '#064e3b', text: '#ffffff' },
    },
  },
  update: {
    key: 'update',
    label: 'UPDATE',
    pillBg: '#2563eb',
    pillText: '#ffffff',
    levels: {
      6: { fill: '#eff6ff', stroke: '#bfdbfe', text: '#1d4ed8' },
      25: { fill: '#dbeafe', stroke: '#93c5fd', text: '#1e40af' },
      42: { fill: '#bfdbfe', stroke: '#60a5fa', text: '#1e3a8a' },
      61: { fill: '#93c5fd', stroke: '#3b82f6', text: '#172554' },
      86: { fill: '#60a5fa', stroke: '#2563eb', text: '#0f172a' },
      99: { fill: '#2563eb', stroke: '#1d4ed8', text: '#ffffff' },
      100: { fill: '#1d4ed8', stroke: '#1e3a8a', text: '#ffffff' },
    },
  },
  delete: {
    key: 'delete',
    label: 'DELETE',
    pillBg: '#dc2626',
    pillText: '#ffffff',
    levels: {
      6: { fill: '#fef2f2', stroke: '#fecaca', text: '#b91c1c' },
      25: { fill: '#fee2e2', stroke: '#fca5a5', text: '#991b1b' },
      42: { fill: '#fecaca', stroke: '#f87171', text: '#7f1d1d' },
      61: { fill: '#fca5a5', stroke: '#ef4444', text: '#450a0a' },
      86: { fill: '#f87171', stroke: '#dc2626', text: '#450a0a' },
      99: { fill: '#dc2626', stroke: '#b91c1c', text: '#ffffff' },
      100: { fill: '#b91c1c', stroke: '#7f1d1d', text: '#ffffff' },
    },
  },
};

const HEXBIN_OP_PALETTES_DARK = {
  create: {
    key: 'create',
    label: 'CREATE',
    pillBg: '#059669',
    pillText: '#ffffff',
    levels: {
      6: { fill: '#0b2520', stroke: '#134e40', text: '#6ee7b7' },
      25: { fill: '#0f362d', stroke: '#047857', text: '#a7f3d0' },
      42: { fill: '#114b3c', stroke: '#059669', text: '#d1fae5' },
      61: { fill: '#047857', stroke: '#10b981', text: '#ecfdf5' },
      86: { fill: '#059669', stroke: '#34d399', text: '#ffffff' },
      99: { fill: '#10b981', stroke: '#6ee7b7', text: '#022c22' },
      100: { fill: '#34d399', stroke: '#a7f3d0', text: '#022c22' },
    },
  },
  update: {
    key: 'update',
    label: 'UPDATE',
    pillBg: '#2563eb',
    pillText: '#ffffff',
    levels: {
      6: { fill: '#101f38', stroke: '#1e3a6e', text: '#93c5fd' },
      25: { fill: '#162d54', stroke: '#1d4ed8', text: '#bfdbfe' },
      42: { fill: '#1d3e75', stroke: '#2563eb', text: '#dbeafe' },
      61: { fill: '#1d4ed8', stroke: '#3b82f6', text: '#eff6ff' },
      86: { fill: '#2563eb', stroke: '#60a5fa', text: '#ffffff' },
      99: { fill: '#3b82f6', stroke: '#93c5fd', text: '#ffffff' },
      100: { fill: '#60a5fa', stroke: '#bfdbfe', text: '#0f172a' },
    },
  },
  delete: {
    key: 'delete',
    label: 'DELETE',
    pillBg: '#dc2626',
    pillText: '#ffffff',
    levels: {
      6: { fill: '#2a1317', stroke: '#5c1d24', text: '#fca5a5' },
      25: { fill: '#3f181e', stroke: '#7f1d1d', text: '#fecaca' },
      42: { fill: '#5c1d24', stroke: '#b91c1c', text: '#fee2e2' },
      61: { fill: '#991b1b', stroke: '#dc2626', text: '#fef2f2' },
      86: { fill: '#dc2626', stroke: '#ef4444', text: '#ffffff' },
      99: { fill: '#ef4444', stroke: '#f87171', text: '#ffffff' },
      100: { fill: '#f87171', stroke: '#fecaca', text: '#450a0a' },
    },
  },
};

function buildHexbinBandCells(seedStr, rows = 10, cols = 16, opWeights = {}) {
  let seed = 2166136261;
  for (let i = 0; i < seedStr.length; i++) {
    seed ^= seedStr.charCodeAt(i);
    seed = Math.imul(seed, 16777619) >>> 0;
  }

  const baseCreate = Math.max(1, opWeights.createCount || 12);
  const baseUpdate = Math.max(1, opWeights.updateCount || 10);
  const baseDelete = Math.max(1, opWeights.deleteCount || 6);
  const totalWeight = baseCreate + baseUpdate + baseDelete;
  const wCreate = 0.55 + (baseCreate / totalWeight);
  const wUpdate = 0.55 + (baseUpdate / totalWeight);
  const wDelete = 0.55 + (baseDelete / totalWeight);

  const safeRows = Math.max(3, rows);
  const safeCols = Math.max(4, cols);
  const createPeakR = Math.min(safeRows - 2, Math.max(1, (seed >>> 3) % (safeRows - 1)));
  const createPeakC = Math.min(safeCols - 2, Math.max(1, (seed >>> 7) % (safeCols - 2)));
  const updatePeakR = Math.min(safeRows - 2, Math.max(1, (createPeakR + Math.floor(safeRows / 3)) % (safeRows - 1)));
  const updatePeakC = Math.min(safeCols - 2, Math.max(1, (createPeakC + Math.floor(safeCols / 3)) % (safeCols - 1)));
  const deletePeakR = Math.min(safeRows - 2, Math.max(1, (createPeakR + Math.floor((safeRows * 2) / 3)) % (safeRows - 1)));
  const deletePeakC = Math.min(safeCols - 2, Math.max(1, (createPeakC + Math.floor((safeCols * 2) / 3)) % (safeCols - 1)));

  const cells = [];
  for (let r = 0; r < rows; r++) {
    for (let c = 0; c < cols; c++) {
      seed ^= (r * 37 + c * 23 + 19) & 0xff;
      seed = Math.imul(seed, 16777619) >>> 0;
      const h1 = seed >>> 0;
      seed ^= (r * 53 + c * 41 + 7) & 0xff;
      seed = Math.imul(seed, 16777619) >>> 0;
      const h2 = seed >>> 0;

      const dCreate = Math.hypot(r - createPeakR, c - createPeakC);
      const dUpdate = Math.hypot(r - updatePeakR, c - updatePeakC);
      const dDelete = Math.hypot(r - deletePeakR, c - deletePeakC);

      const boostCreate = Math.max(0, 14 - dCreate * 3.1) * 4;
      const boostUpdate = Math.max(0, 14 - dUpdate * 3.1) * 4;
      const boostDelete = Math.max(0, 14 - dDelete * 3.1) * 4;

      const cellAdded = Math.max(0, Math.round(((h1 % 18) + boostCreate) * wCreate));
      const cellModified = Math.max(0, Math.round((((h1 >>> 5) % 18) + boostUpdate) * wUpdate));
      const cellDeleted = Math.max(0, Math.round((((h2 >>> 3) % 17) + boostDelete) * wDelete));

      let op = 'create';
      let maxOpVal = cellAdded;
      if (cellModified > maxOpVal) {
        op = 'update';
        maxOpVal = cellModified;
      }
      if (cellDeleted > maxOpVal) {
        op = 'delete';
        maxOpVal = cellDeleted;
      }

      const minDist = Math.min(dCreate, dUpdate, dDelete);
      let pct = 6;
      if (
        (r === createPeakR && c === createPeakC) ||
        (r === updatePeakR && c === updatePeakC) ||
        (r === deletePeakR && c === deletePeakC)
      ) {
        pct = 100;
      } else if (minDist <= 1.35) {
        pct = (h1 & 1) === 0 ? 99 : 86;
      } else if (minDist <= 2.5) {
        const pick = h1 % 3;
        pct = pick === 0 ? 86 : pick === 1 ? 61 : 42;
      } else {
        const pick = (h1 + h2) % 5;
        pct = pick === 0 ? 61 : pick === 1 ? 42 : pick === 2 ? 25 : 6;
      }

      cells.push({
        row: r,
        col: c,
        pct,
        op,
        added: cellAdded,
        modified: cellModified,
        deleted: cellDeleted,
      });
    }
  }
  return cells;
}

function compareValues(aVal, bVal, dir = 'asc') {
  const mult = dir === 'desc' ? -1 : 1;
  if (typeof aVal === 'number' && typeof bVal === 'number') {
    return (aVal - bVal) * mult;
  }
  const aStr = String(aVal ?? '');
  const bStr = String(bVal ?? '');
  return (
    aStr.localeCompare(bStr, undefined, { numeric: true, sensitivity: 'base' }) *
    mult
  );
}

function getTreeNodeSortValue(node, field) {
  switch (field) {
    case 'name':
      return node.name || '';
    case 'pctOfParent':
      return node.pctOfParent || 0;
    case 'pctOfRoot':
      return node.pctOfRoot || 0;
    case 'subtreeBytes':
      return node.subtreeBytes || 0;
    case 'fileCount':
      return node.isDir ? node.fileCount || 0 : 1;
    case 'dirCount':
      return node.isDir ? node.dirCount || 0 : -1;
    case 'hash':
      return node.kind === 'root' && node.driveId ? node.driveId : node.hash || '';
    default:
      return node.subtreeBytes || 0;
  }
}

function flattenVisibleTree(node, expandedMap, treeSort, depth = 0, out = []) {
  if (!node) return out;
  out.push({ node, depth });
  if (node.isDir && expandedMap[node.key]) {
    const children = [...(node.children || [])];
    if (treeSort && treeSort.field) {
      children.sort((a, b) => {
        const cmp = compareValues(
          getTreeNodeSortValue(a, treeSort.field),
          getTreeNodeSortValue(b, treeSort.field),
          treeSort.dir
        );
        if (cmp !== 0) return cmp;
        return compareValues(a.name || '', b.name || '', 'asc');
      });
    }
    for (const ch of children) {
      flattenVisibleTree(ch, expandedMap, treeSort, depth + 1, out);
    }
  }
  return out;
}

function collectDirectoryNodes(node, depth = 0, out = []) {
  if (!node || !node.isDir) return out;
  out.push({ node, depth });
  for (const ch of node.children || []) {
    if (ch.isDir) {
      collectDirectoryNodes(ch, depth + 1, out);
    }
  }
  return out;
}

function computeTreemapLayout(items, x = 0, y = 0, w = 100, h = 100) {
  if (!items || items.length === 0 || w <= 0 || h <= 0) return [];
  if (items.length === 1) {
    return [{ item: items[0], x, y, w, h }];
  }

  const total = items.reduce((acc, it) => acc + Math.max(1, it.subtreeBytes || it.size || 1), 0);
  let half = 0;
  let splitIdx = 0;
  for (let i = 0; i < items.length; i++) {
    const weight = Math.max(1, items[i].subtreeBytes || items[i].size || 1);
    if (half + weight > total / 2 && i > 0) {
      splitIdx = i;
      break;
    }
    half += weight;
    splitIdx = i + 1;
  }
  if (splitIdx <= 0) splitIdx = 1;
  if (splitIdx >= items.length) splitIdx = items.length - 1;

  const leftItems = items.slice(0, splitIdx);
  const rightItems = items.slice(splitIdx);
  const leftWeight = leftItems.reduce((acc, it) => acc + Math.max(1, it.subtreeBytes || it.size || 1), 0);
  const ratio = total > 0 ? leftWeight / total : 0.5;

  if (w >= h) {
    const wLeft = w * ratio;
    return [
      ...computeTreemapLayout(leftItems, x, y, wLeft, h),
      ...computeTreemapLayout(rightItems, x + wLeft, y, w - wLeft, h),
    ];
  }
  const hTop = h * ratio;
  return [
    ...computeTreemapLayout(leftItems, x, y, w, hTop),
    ...computeTreemapLayout(rightItems, x, y + hTop, w, h - hTop),
  ];
}

export default function HomePage() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [sidebarExpanded, setSidebarExpanded] = useState(false);
  const [darkMode, setDarkMode] = useState(false);
  const [serverOnline, setServerOnline] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  // Scans & Computers state
  const [scans, setScans] = useState([]);
  const [computers, setComputers] = useState([]);
  const [loadingScans, setLoadingScans] = useState(true);
  const [selectedScanId, setSelectedScanId] = useState('');
  const [selectedScan, setSelectedScan] = useState(null);
  const [detailView, setDetailView] = useState('entries'); // 'entries' | 'history'
  const [entryFilter, setEntryFilter] = useState('');
  const [scanComputerFilter, setScanComputerFilter] = useState('');
  const [scanEntryKindFilter, setScanEntryKindFilter] = useState('all'); // 'all' | 'files' | 'dirs'

  // Diff state
  const [baseScanId, setBaseScanId] = useState('');
  const [targetScanId, setTargetScanId] = useState('');
  const [diffReport, setDiffReport] = useState(null);
  const [loadingDiff, setLoadingDiff] = useState(false);
  const [diffChangeFilter, setDiffChangeFilter] = useState('all'); // 'all' | 'added' | 'modified' | 'deleted'

  // Duplicates state
  const [dupScanFilter, setDupScanFilter] = useState('');
  const [duplicates, setDuplicates] = useState([]);
  const [loadingDups, setLoadingDups] = useState(false);
  const [selectedDupKey, setSelectedDupKey] = useState('');

  // WinDirStat Tree & Treemap state
  const [selectedComputerId, setSelectedComputerId] = useState('');
  const [scanDetailsMap, setScanDetailsMap] = useState({});
  const [loadingTree, setLoadingTree] = useState(false);
  const [expandedTreeNodes, setExpandedTreeNodes] = useState({});
  const [focusedTreeKey, setFocusedTreeKey] = useState('');
  const [highlightedExt, setHighlightedExt] = useState('');
  const [selectedTreemapFile, setSelectedTreemapFile] = useState(null);
  const [treemapMode, setTreemapMode] = useState('modules'); // 'modules' | 'files'
  const attemptedScanIdsRef = useRef(new Set());

  // File Types Distribution state
  const [fileTypeScope, setFileTypeScope] = useState('__all__');
  const [fileTypeMetric, setFileTypeMetric] = useState('bytes'); // 'bytes' | 'count'
  const [selectedFileTypeExt, setSelectedFileTypeExt] = useState('');
  const [opDrawerOpen, setOpDrawerOpen] = useState(true);
  const [showTreemapLegend, setShowTreemapLegend] = useState(false);
  const [isoZoom, setIsoZoom] = useState(1);
  const [isoPan, setIsoPan] = useState({ x: 0, y: 0 });
  const [copiedKey, setCopiedKey] = useState('');
  const [hexbinPeriod, setHexbinPeriod] = useState('day'); // 'day' | 'week' | 'month' | 'year'
  const hexbinBandsRef = useRef(null);
  const [hexbinBox, setHexbinBox] = useState({ width: 920, height: 620 });
  const [tableSorts, setTableSorts] = useState({
    dashScans: { field: 'version', dir: 'desc' },
    dashDups: { field: 'wasted_bytes', dir: 'desc' },
    tree: { field: 'subtreeBytes', dir: 'desc' },
    treeExts: { field: 'bytes', dir: 'desc' },
    ftStats: { field: 'bytes', dir: 'desc' },
    ftFiles: { field: 'size', dir: 'desc' },
    scanEntries: { field: 'rel_path', dir: 'asc' },
    scanHistory: { field: 'rel_path', dir: 'asc' },
    diff: { field: 'rel_path', dir: 'asc' },
    dups: { field: 'wasted_bytes', dir: 'desc' },
  });
  const searchInputRef = useRef(null);
  const panDragRef = useRef({ active: false, startX: 0, startY: 0, origX: 0, origY: 0, moved: false });

  const handleSortTable = useCallback((tableId, field, defaultDir = 'desc') => {
    setTableSorts((prev) => {
      const current = prev[tableId] || { field: '', dir: defaultDir };
      if (current.field === field) {
        return {
          ...prev,
          [tableId]: {
            field,
            dir: current.dir === 'asc' ? 'desc' : 'asc',
          },
        };
      }
      return {
        ...prev,
        [tableId]: { field, dir: defaultDir },
      };
    });
  }, []);

  const renderSortableTh = (tableId, field, label, defaultDir = 'asc') => {
    const current = tableSorts[tableId];
    const isActive = current?.field === field;
    const dir = isActive ? current.dir : null;
    return (
      <th aria-sort={isActive ? (dir === 'asc' ? 'ascending' : 'descending') : 'none'}>
        <button
          type="button"
          className={`th-sort-btn ${isActive ? 'active' : ''}`}
          onClick={() => handleSortTable(tableId, field, defaultDir)}
          title={`Sort by ${label}`}
        >
          <span>{label}</span>
          <span className="th-sort-icon" aria-hidden="true">
            {isActive ? (dir === 'asc' ? '↑' : '↓') : '↕'}
          </span>
        </button>
      </th>
    );
  };

  const handleCopyText = useCallback((text, key) => {
    if (!text) return;
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(text).catch(() => {});
    }
    setCopiedKey(key);
    setTimeout(() => {
      setCopiedKey((prev) => (prev === key ? '' : prev));
    }, 1600);
  }, []);

  useEffect(() => {
    try {
      const saved = window.localStorage.getItem('gols-theme');
      if (saved === 'dark') {
        setDarkMode(true);
      } else if (saved === 'light') {
        setDarkMode(false);
      }
    } catch {
      // ignore storage errors
    }
  }, []);

  const handleToggleTheme = useCallback(() => {
    setDarkMode((prev) => {
      const next = !prev;
      try {
        window.localStorage.setItem('gols-theme', next ? 'dark' : 'light');
      } catch {
        // ignore storage errors
      }
      return next;
    });
  }, []);

  useEffect(() => {
    const onKeyDown = (e) => {
      if (
        e.key === '/' &&
        !e.metaKey &&
        !e.ctrlKey &&
        !e.altKey &&
        document.activeElement?.tagName !== 'INPUT' &&
        document.activeElement?.tagName !== 'TEXTAREA' &&
        document.activeElement?.tagName !== 'SELECT'
      ) {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  useEffect(() => {
    if (activeTab !== 'dashboard') return undefined;
    const node = hexbinBandsRef.current;
    if (!node || typeof ResizeObserver === 'undefined') return undefined;
    const measure = () => {
      const rect = node.getBoundingClientRect();
      if (rect.width > 80 && rect.height > 80) {
        setHexbinBox((prev) => {
          const w = Math.round(rect.width);
          const h = Math.round(rect.height);
          if (Math.abs(prev.width - w) < 3 && Math.abs(prev.height - h) < 3) return prev;
          return { width: w, height: h };
        });
      }
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(node);
    return () => ro.disconnect();
  }, [activeTab, scans.length, entryFilter, selectedComputerId]);

  const fetchScans = useCallback(async () => {
    setLoadingScans(true);
    setErrorMsg('');
    attemptedScanIdsRef.current.clear();
    try {
      const healthRes = await fetch('/healthz');
      setServerOnline(healthRes.ok);

      const [scansRes, compsRes, dupsRes] = await Promise.all([
        fetch('/api/v1/scans'),
        fetch('/api/v1/computers'),
        fetch('/api/v1/duplicates'),
      ]);
      if (!scansRes.ok) throw new Error(`Failed to load scans (${scansRes.status})`);
      const data = await scansRes.json();
      const list = Array.isArray(data) ? data : [];
      setScans(list);

      let compList = [];
      if (compsRes.ok) {
        const compData = await compsRes.json();
        compList = Array.isArray(compData) ? compData : [];
        setComputers(compList);
      }
      if (dupsRes.ok) {
        const dupData = await dupsRes.json();
        setDuplicates(Array.isArray(dupData) ? dupData : []);
      }

      if (compList.length > 0) {
        setSelectedComputerId((prev) => prev || compList[0].id);
      }
      if (list.length > 0) {
        setSelectedScanId((prev) => prev || list[0].id);
      }
      if (list.length >= 2) {
        setBaseScanId((prev) => prev || list[list.length - 1].id);
        setTargetScanId((prev) => prev || list[0].id);
      } else if (list.length === 1) {
        setBaseScanId((prev) => prev || list[0].id);
        setTargetScanId((prev) => prev || list[0].id);
      }
    } catch (err) {
      setServerOnline(false);
      setErrorMsg(err.message || 'Unable to reach go-ls server API.');
    } finally {
      setLoadingScans(false);
    }
  }, []);

  useEffect(() => {
    fetchScans();
  }, [fetchScans]);

  useEffect(() => {
    if (!selectedScanId) {
      setSelectedScan(null);
      return;
    }
    let cancelled = false;
    async function loadScanDetail() {
      try {
        const res = await fetch(`/api/v1/scans/${encodeURIComponent(selectedScanId)}`);
        if (!res.ok) throw new Error(`Could not load scan ${selectedScanId}`);
        const detail = await res.json();
        if (!cancelled && detail && detail.id) {
          attemptedScanIdsRef.current.add(detail.id);
          setSelectedScan(detail);
          setScanDetailsMap((prev) =>
            prev[detail.id] === detail ? prev : { ...prev, [detail.id]: detail }
          );
        }
      } catch (err) {
        if (!cancelled) setErrorMsg(err.message);
      }
    }
    loadScanDetail();
    return () => {
      cancelled = true;
    };
  }, [selectedScanId]);

  const effectiveComputers = useMemo(() => {
    if (computers.length > 0) return computers;
    if (scans.length > 0) {
      return [
        {
          id: 'default-host',
          hostname: scans[0].hostname || 'Local Host',
          drives: Array.from(new Set(scans.map((s) => s.drive_id).filter(Boolean))),
          directories: scans,
          total_files: scans.reduce((a, s) => a + (s.total_files || 0), 0),
          total_dirs: scans.reduce((a, s) => a + (s.total_dirs || 0), 0),
          total_bytes: scans.reduce((a, s) => a + (s.total_bytes || 0), 0),
        },
      ];
    }
    return [];
  }, [computers, scans]);

  const activeComputer = useMemo(() => {
    if (effectiveComputers.length === 0) return null;
    return (
      effectiveComputers.find((c) => c.id === selectedComputerId) || effectiveComputers[0]
    );
  }, [effectiveComputers, selectedComputerId]);

  useEffect(() => {
    const targetDirs =
      activeTab === 'filetypes' || activeTab === 'dashboard'
        ? scans
        : activeComputer
          ? activeComputer.directories || []
          : [];
    const missingIds = targetDirs
      .map((d) => d.id)
      .filter((id) => id && !attemptedScanIdsRef.current.has(id));
    if (missingIds.length === 0) return;

    let cancelled = false;
    async function loadComputerScans() {
      setLoadingTree(true);
      try {
        const fetched = await Promise.all(
          missingIds.map(async (id) => {
            const res = await fetch(`/api/v1/scans/${encodeURIComponent(id)}`);
            if (!res.ok) return null;
            return res.json();
          })
        );
        if (cancelled) return;
        const valid = fetched.filter((rec) => rec && rec.id);
        for (const rec of valid) {
          attemptedScanIdsRef.current.add(rec.id);
        }
        if (valid.length > 0) {
          setScanDetailsMap((prev) => {
            const next = { ...prev };
            for (const rec of valid) {
              next[rec.id] = rec;
            }
            return next;
          });
        }
      } catch (err) {
        if (!cancelled) setErrorMsg(err.message);
      } finally {
        if (!cancelled) setLoadingTree(false);
      }
    }
    loadComputerScans();
    return () => {
      cancelled = true;
    };
  }, [activeComputer, activeTab, scans]);

  const winDirRoot = useMemo(
    () => buildWinDirTree(activeComputer, scanDetailsMap),
    [activeComputer, scanDetailsMap]
  );

  useEffect(() => {
    if (!winDirRoot) return;
    setExpandedTreeNodes((prev) => {
      const next = { ...prev, [winDirRoot.key]: true };
      for (const ch of winDirRoot.children || []) {
        if (next[ch.key] === undefined) {
          next[ch.key] = true;
        }
      }
      return next;
    });
    setFocusedTreeKey((prev) => {
      if (prev && findNodeByKey(winDirRoot, prev)) return prev;
      return winDirRoot.key;
    });
  }, [winDirRoot]);

  const focusedTreeNode = useMemo(() => {
    if (!winDirRoot) return null;
    return findNodeByKey(winDirRoot, focusedTreeKey) || winDirRoot;
  }, [winDirRoot, focusedTreeKey]);

  const treeBreadcrumbs = useMemo(() => {
    if (!winDirRoot || !focusedTreeNode) return [];
    return getNodeAncestors(winDirRoot, focusedTreeNode.key);
  }, [winDirRoot, focusedTreeNode]);

  const visibleTreeRows = useMemo(() => {
    if (!winDirRoot) return [];
    return flattenVisibleTree(winDirRoot, expandedTreeNodes, tableSorts.tree);
  }, [winDirRoot, expandedTreeNodes, tableSorts.tree]);

  const allDirectoryNodes = useMemo(() => {
    if (!winDirRoot) return [];
    return collectDirectoryNodes(winDirRoot);
  }, [winDirRoot]);

  const { extensionStats, treemapRects, focusedLeafFiles, isoCushionBlocks } = useMemo(() => {
    if (!focusedTreeNode) {
      return { extensionStats: [], treemapRects: [], focusedLeafFiles: [], isoCushionBlocks: [] };
    }
    const leaves = collectLeafFiles(focusedTreeNode).filter((f) => (f.subtreeBytes || f.size || 0) > 0);
    leaves.sort((a, b) => (b.subtreeBytes || b.size || 0) - (a.subtreeBytes || a.size || 0));

    const totalFocusedBytes = leaves.reduce((acc, f) => acc + (f.subtreeBytes || f.size || 0), 0);
    const byExt = new Map();
    for (const f of leaves) {
      const ext = f.ext || getFileExtension(f.name);
      const prev = byExt.get(ext) || { ext, bytes: 0, count: 0 };
      prev.bytes += f.subtreeBytes || f.size || 0;
      prev.count += 1;
      byExt.set(ext, prev);
    }

    const extSort = tableSorts.treeExts || { field: 'bytes', dir: 'desc' };
    const extList = Array.from(byExt.values())
      .map((item) => ({
        ...item,
        pct: totalFocusedBytes > 0 ? (item.bytes / totalFocusedBytes) * 100 : 0,
        style: getExtensionStyle(item.ext),
      }))
      .sort((a, b) => {
        switch (extSort.field) {
          case 'ext':
            return compareValues(a.ext, b.ext, extSort.dir);
          case 'color':
            return compareValues(a.style?.swatch || a.ext, b.style?.swatch || b.ext, extSort.dir);
          case 'pct':
            return compareValues(a.pct, b.pct, extSort.dir);
          case 'count':
            return compareValues(a.count, b.count, extSort.dir);
          case 'bytes':
          default:
            return compareValues(a.bytes, b.bytes, extSort.dir);
        }
      });

    const topLeavesForTreemap = leaves.slice(0, 140);
    const rects = computeTreemapLayout(topLeavesForTreemap, 0, 0, 100, 100);

    let moduleCandidates = [];
    if (treemapMode === 'modules') {
      let cursor = focusedTreeNode;
      while (
        cursor &&
        cursor.isDir &&
        Array.isArray(cursor.children) &&
        cursor.children.length === 1 &&
        cursor.children[0].isDir &&
        (cursor.children[0].children || []).length > 0
      ) {
        cursor = cursor.children[0];
      }
      const rawChildren = (cursor?.children || []).filter(
        (ch) => (ch.subtreeBytes || ch.size || 0) > 0
      );
      moduleCandidates =
        rawChildren.length > 0 ? rawChildren.slice(0, 12) : leaves.slice(0, 12);
    } else {
      moduleCandidates = leaves.slice(0, 16);
    }

    const totalBlockBytes = moduleCandidates.reduce(
      (acc, it) => acc + Math.max(1, it.subtreeBytes || it.size || 1),
      0
    );
    const maxBlockBytes = moduleCandidates.reduce(
      (acc, it) => Math.max(acc, it.subtreeBytes || it.size || 1),
      1
    );
    const rawIsoRects = computeTreemapLayout(moduleCandidates, 0, 0, 100, 100);
    const isoBlocks = rawIsoRects.map((r, idx) => {
      const bytes = r.item.subtreeBytes || r.item.size || 0;
      const pct = totalBlockBytes > 0 ? (bytes / totalBlockBytes) * 100 : 0;
      const ratioToMax = maxBlockBytes > 0 ? bytes / maxBlockBytes : 0.5;
      const litDots = Math.max(1, Math.min(5, Math.round(ratioToMax * 4) + 1));
      const dominantExt = r.item.isDir
        ? collectLeafFiles(r.item)[0]?.ext || '<dir>'
        : r.item.ext || getFileExtension(r.item.name);
      const style = getExtensionStyle(dominantExt);
      return {
        ...r,
        index: idx + 1,
        bytes,
        pct,
        ratioToMax,
        litDots,
        dominantExt,
        style,
      };
    });

    return {
      extensionStats: extList,
      treemapRects: rects,
      focusedLeafFiles: leaves,
      isoCushionBlocks: isoBlocks,
    };
  }, [focusedTreeNode, treemapMode, tableSorts.treeExts]);

  const fileTypeAnalysis = useMemo(() => {
    let targetScanSummaries = scans;
    if (fileTypeScope.startsWith('comp:')) {
      const compId = fileTypeScope.slice(5);
      const comp = effectiveComputers.find((c) => c.id === compId);
      targetScanSummaries = comp ? comp.directories || [] : [];
    } else if (fileTypeScope.startsWith('scan:')) {
      const sId = fileTypeScope.slice(5);
      targetScanSummaries = scans.filter((s) => s.id === sId);
    }

    const allFiles = [];
    for (const s of targetScanSummaries) {
      const detail = scanDetailsMap[s.id];
      if (!detail || !Array.isArray(detail.entries)) continue;
      for (const entry of detail.entries) {
        if (entry.is_dir || !entry.rel_path || entry.rel_path === '.') continue;
        const name = entry.name || entry.rel_path.split('/').pop() || entry.rel_path;
        const ext = getFileExtension(name);
        allFiles.push({
          key: `${s.id}:${entry.rel_path}`,
          scanId: s.id,
          hostname: s.hostname,
          driveId: s.drive_id || 'default-drive',
          rootPath: s.root_path,
          relPath: entry.rel_path,
          absPath: entry.abs_path || `${s.root_path}/${entry.rel_path}`,
          name,
          ext,
          category: getFileCategory(ext),
          size: entry.size || 0,
          hash: entry.hash || '',
        });
      }
    }

    allFiles.sort((a, b) => b.size - a.size);
    const totalBytes = allFiles.reduce((acc, f) => acc + f.size, 0);
    const totalCount = allFiles.length;

    const byExt = new Map();
    for (const f of allFiles) {
      const prev = byExt.get(f.ext) || {
        ext: f.ext,
        category: f.category,
        bytes: 0,
        count: 0,
        largestFile: f,
      };
      prev.bytes += f.size;
      prev.count += 1;
      if (f.size > (prev.largestFile?.size || 0)) {
        prev.largestFile = f;
      }
      byExt.set(f.ext, prev);
    }

    const q = entryFilter.trim().toLowerCase();
    const ftSort = tableSorts.ftStats || { field: 'bytes', dir: 'desc' };
    const stats = Array.from(byExt.values())
      .map((item) => ({
        ...item,
        pctBytes: totalBytes > 0 ? (item.bytes / totalBytes) * 100 : 0,
        pctCount: totalCount > 0 ? (item.count / totalCount) * 100 : 0,
        avgBytes: item.count > 0 ? Math.round(item.bytes / item.count) : 0,
        style: getExtensionStyle(item.ext),
      }))
      .filter((item) => {
        if (!q || activeTab !== 'filetypes') return true;
        return (
          item.ext.toLowerCase().includes(q) ||
          item.category.toLowerCase().includes(q) ||
          item.largestFile?.name?.toLowerCase().includes(q)
        );
      })
      .sort((a, b) => {
        switch (ftSort.field) {
          case 'ext':
            return compareValues(a.ext, b.ext, ftSort.dir);
          case 'category':
            return compareValues(a.category, b.category, ftSort.dir);
          case 'pct':
            return compareValues(
              fileTypeMetric === 'count' ? a.pctCount : a.pctBytes,
              fileTypeMetric === 'count' ? b.pctCount : b.pctBytes,
              ftSort.dir
            );
          case 'count':
            return compareValues(a.count, b.count, ftSort.dir);
          case 'avgBytes':
            return compareValues(a.avgBytes, b.avgBytes, ftSort.dir);
          case 'largestFile':
            return compareValues(a.largestFile?.size || 0, b.largestFile?.size || 0, ftSort.dir);
          case 'bytes':
          default:
            return compareValues(a.bytes, b.bytes, ftSort.dir);
        }
      });

    let matchingFiles = selectedFileTypeExt
      ? allFiles.filter((f) => f.ext === selectedFileTypeExt)
      : allFiles;
    if (q && activeTab === 'filetypes') {
      matchingFiles = matchingFiles.filter(
        (f) =>
          f.name.toLowerCase().includes(q) ||
          f.absPath.toLowerCase().includes(q) ||
          f.ext.toLowerCase().includes(q) ||
          f.category.toLowerCase().includes(q) ||
          f.hash.toLowerCase().includes(q)
      );
    }
    const ftFileSort = tableSorts.ftFiles || { field: 'size', dir: 'desc' };
    matchingFiles = [...matchingFiles]
      .sort((a, b) => {
        switch (ftFileSort.field) {
          case 'path':
            return compareValues(`${a.hostname}:${a.absPath}`, `${b.hostname}:${b.absPath}`, ftFileSort.dir);
          case 'ext':
            return compareValues(a.ext, b.ext, ftFileSort.dir);
          case 'hostname':
            return compareValues(a.hostname, b.hostname, ftFileSort.dir);
          case 'size':
          default:
            return compareValues(a.size || 0, b.size || 0, ftFileSort.dir);
        }
      })
      .slice(0, 100);

    return {
      stats,
      matchingFiles,
      totalBytes,
      totalCount,
    };
  }, [scans, effectiveComputers, scanDetailsMap, fileTypeScope, fileTypeMetric, selectedFileTypeExt, entryFilter, activeTab, tableSorts.ftStats, tableSorts.ftFiles]);

  const handleToggleTreeExpand = (nodeKey, e) => {
    if (e) e.stopPropagation();
    setExpandedTreeNodes((prev) => ({
      ...prev,
      [nodeKey]: !prev[nodeKey],
    }));
  };

  const handleExpandAllTree = () => {
    if (!winDirRoot) return;
    const all = {};
    function walk(n) {
      if (n.isDir) {
        all[n.key] = true;
        for (const ch of n.children || []) walk(ch);
      }
    }
    walk(winDirRoot);
    setExpandedTreeNodes(all);
  };

  const handleCollapseTreeToRoots = () => {
    if (!winDirRoot) return;
    const minimal = { [winDirRoot.key]: true };
    for (const ch of winDirRoot.children || []) {
      minimal[ch.key] = true;
    }
    setExpandedTreeNodes(minimal);
    setFocusedTreeKey(winDirRoot.key);
  };

  const runDiffRequest = useCallback(async (baseId, targetId) => {
    if (!baseId || !targetId) return;
    setLoadingDiff(true);
    setErrorMsg('');
    try {
      const params = new URLSearchParams({ base: baseId, target: targetId });
      const res = await fetch(`/api/v1/diff?${params.toString()}`);
      if (!res.ok) throw new Error(`Diff request failed (${res.status})`);
      const data = await res.json();
      setDiffReport(data);
    } catch (err) {
      setErrorMsg(err.message);
    } finally {
      setLoadingDiff(false);
    }
  }, []);

  const handleRunDiff = (e) => {
    if (e) e.preventDefault();
    runDiffRequest(baseScanId, targetScanId);
  };

  useEffect(() => {
    if (activeTab === 'diff' && baseScanId && targetScanId) {
      runDiffRequest(baseScanId, targetScanId);
    }
  }, [activeTab, baseScanId, targetScanId, runDiffRequest]);

  const fetchDuplicates = useCallback(async (scanId) => {
    setLoadingDups(true);
    setErrorMsg('');
    try {
      const query = scanId ? `?scan_id=${encodeURIComponent(scanId)}` : '';
      const res = await fetch(`/api/v1/duplicates${query}`);
      if (!res.ok) throw new Error(`Duplicates request failed (${res.status})`);
      const data = await res.json();
      setDuplicates(Array.isArray(data) ? data : []);
    } catch (err) {
      setErrorMsg(err.message);
    } finally {
      setLoadingDups(false);
    }
  }, []);

  useEffect(() => {
    if (activeTab === 'duplicates') {
      fetchDuplicates(dupScanFilter);
    }
  }, [activeTab, dupScanFilter, fetchDuplicates]);

  const totalFilesAcrossScans = scans.reduce((acc, s) => acc + (s.total_files || 0), 0);
  const totalBytesAcrossScans = scans.reduce((acc, s) => acc + (s.total_bytes || 0), 0);
  const totalWastedBytes = duplicates.reduce((acc, g) => acc + (g.wasted_bytes || 0), 0);

  const activeComputerScans = useMemo(() => {
    if (!activeComputer) return [];
    return scans.filter(
      (s) =>
        s.computer_id === activeComputer.id ||
        s.hostname === activeComputer.hostname ||
        s.hostname === activeComputer.id
    );
  }, [scans, activeComputer]);

  const activeComputerFiles = activeComputerScans.reduce((acc, s) => acc + (s.total_files || 0), 0);
  const activeComputerDirs = activeComputerScans.reduce((acc, s) => acc + (s.total_dirs || 0), 0);
  const activeComputerBytes = activeComputerScans.reduce((acc, s) => acc + (s.total_bytes || 0), 0);
  const activeComputerDrivesCount = (activeComputer?.drives || []).length;

  const activeComputerDuplicates = useMemo(() => {
    if (!activeComputer) return [];
    return duplicates.filter((g) =>
      (g.files || []).some(
        (f) => f.hostname === activeComputer.hostname || f.computer_id === activeComputer.id
      )
    );
  }, [duplicates, activeComputer]);

  const activeComputerWastedBytes = activeComputerDuplicates.reduce(
    (acc, g) => acc + (g.wasted_bytes || 0),
    0
  );

  const dashboardFileTypeStats = useMemo(() => {
    const byExt = new Map();
    let totalBytes = 0;
    for (const s of activeComputerScans) {
      const detail = scanDetailsMap[s.id];
      if (!detail || !Array.isArray(detail.entries)) continue;
      for (const entry of detail.entries) {
        if (entry.is_dir || !entry.rel_path || entry.rel_path === '.') continue;
        const name = entry.name || entry.rel_path.split('/').pop() || entry.rel_path;
        const ext = getFileExtension(name);
        const size = entry.size || 0;
        totalBytes += size;
        const prev = byExt.get(ext) || {
          ext,
          category: getFileCategory(ext),
          bytes: 0,
          count: 0,
        };
        prev.bytes += size;
        prev.count += 1;
        byExt.set(ext, prev);
      }
    }
    return Array.from(byExt.values())
      .map((item) => ({
        ...item,
        pctBytes: totalBytes > 0 ? (item.bytes / totalBytes) * 100 : 0,
        style: getExtensionStyle(item.ext),
      }))
      .sort((a, b) => b.bytes - a.bytes);
  }, [activeComputerScans, scanDetailsMap]);

  const activeHexbinPalettes = darkMode ? HEXBIN_OP_PALETTES_DARK : HEXBIN_OP_PALETTES;

  const activeComputerOpSummary = useMemo(() => {
    let createCount = 0;
    let updateCount = 0;
    let deleteCount = 0;
    for (const s of activeComputerScans) {
      const hist = scanDetailsMap[s.id]?.history || [];
      const deltaRevisions = hist.filter((h) => (h.version || 1) > 1);
      if (deltaRevisions.length > 0) {
        for (const h of deltaRevisions) {
          createCount += h.added || 0;
          updateCount += h.modified || 0;
          deleteCount += h.deleted || 0;
        }
      } else if (hist.length > 0) {
        createCount += hist[0].added || s.total_files || 0;
      } else {
        createCount += s.total_files || 0;
      }
    }
    let predominantOp = 'create';
    if (deleteCount >= createCount && deleteCount >= updateCount && deleteCount > 0) {
      predominantOp = 'delete';
    } else if (updateCount >= createCount && updateCount > 0) {
      predominantOp = 'update';
    }
    return {
      createCount,
      updateCount,
      deleteCount,
      predominantOp,
      palette: activeHexbinPalettes[predominantOp] || activeHexbinPalettes.create,
    };
  }, [activeComputerScans, scanDetailsMap, activeHexbinPalettes]);

  const filteredDashboardComputers = useMemo(() => {
    if (!activeComputer) return [];
    if (!entryFilter.trim()) return [activeComputer];
    const q = entryFilter.trim().toLowerCase();
    return [activeComputer].filter(
      (c) =>
        c.hostname?.toLowerCase().includes(q) ||
        c.id?.toLowerCase().includes(q) ||
        (c.drives || []).some((d) => d.toLowerCase().includes(q)) ||
        (c.directories || []).some(
          (dir) =>
            dir.root_path?.toLowerCase().includes(q) ||
            dir.root_hash?.toLowerCase().includes(q)
        )
    );
  }, [activeComputer, entryFilter]);

  const hexbinVisibleScans = useMemo(() => {
    const q = entryFilter.trim().toLowerCase();
    const list = !q
      ? [...activeComputerScans]
      : activeComputerScans.filter(
          (s) =>
            s.hostname?.toLowerCase().includes(q) ||
            s.root_path?.toLowerCase().includes(q) ||
            s.drive_id?.toLowerCase().includes(q) ||
            s.root_hash?.toLowerCase().includes(q) ||
            s.id?.toLowerCase().includes(q)
        );
    const sortCfg = tableSorts.dashScans || { field: 'version', dir: 'desc' };
    return list.sort((a, b) => {
      switch (sortCfg.field) {
        case 'path':
          return compareValues(`${a.hostname}:${a.root_path}`, `${b.hostname}:${b.root_path}`, sortCfg.dir);
        case 'drive_id':
          return compareValues(a.drive_id || 'default-drive', b.drive_id || 'default-drive', sortCfg.dir);
        case 'root_hash':
          return compareValues(a.root_hash || '', b.root_hash || '', sortCfg.dir);
        case 'total_bytes':
          return compareValues(a.total_bytes || 0, b.total_bytes || 0, sortCfg.dir);
        case 'total_files':
          return compareValues(a.total_files || 0, b.total_files || 0, sortCfg.dir);
        case 'version':
        default:
          return compareValues(a.version || 1, b.version || 1, sortCfg.dir);
      }
    });
  }, [activeComputerScans, entryFilter, tableSorts.dashScans]);

  const filteredScansList = useMemo(() => {
    return scans.filter((s) => {
      if (scanComputerFilter && (s.computer_id || s.hostname) !== scanComputerFilter && s.hostname !== scanComputerFilter) {
        return false;
      }
      if (!entryFilter.trim()) return true;
      const q = entryFilter.trim().toLowerCase();
      if (
        s.hostname?.toLowerCase().includes(q) ||
        s.root_path?.toLowerCase().includes(q) ||
        s.drive_id?.toLowerCase().includes(q) ||
        s.root_hash?.toLowerCase().includes(q) ||
        s.id?.toLowerCase().includes(q)
      ) {
        return true;
      }
      const detail = scanDetailsMap[s.id];
      if (detail && Array.isArray(detail.entries)) {
        return detail.entries.some(
          (e) => e.rel_path?.toLowerCase().includes(q) || e.hash?.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [scans, scanComputerFilter, entryFilter, scanDetailsMap]);

  const filteredVisibleTreeRows = useMemo(() => {
    const q = entryFilter.trim().toLowerCase();
    if (!q && !highlightedExt) return visibleTreeRows;
    return visibleTreeRows.filter(({ node }) => {
      if (node.kind === 'computer' || node.kind === 'root') return true;
      const matchesExt = !highlightedExt || node.isDir || node.ext === highlightedExt;
      const matchesSearch =
        !q ||
        node.name?.toLowerCase().includes(q) ||
        node.relPath?.toLowerCase().includes(q) ||
        node.absPath?.toLowerCase().includes(q) ||
        node.hash?.toLowerCase().includes(q) ||
        node.ext?.toLowerCase().includes(q);
      return matchesExt && matchesSearch;
    });
  }, [visibleTreeRows, entryFilter, highlightedExt]);

  const filteredEntries = useMemo(() => {
    const entries = selectedScan?.entries || [];
    const q = entryFilter.trim().toLowerCase();
    const list = entries.filter((entry) => {
      if (scanEntryKindFilter === 'files' && entry.is_dir) return false;
      if (scanEntryKindFilter === 'dirs' && !entry.is_dir) return false;
      if (!q) return true;
      return (
        entry.rel_path?.toLowerCase().includes(q) ||
        entry.hash?.toLowerCase().includes(q) ||
        entry.owner?.toLowerCase().includes(q)
      );
    });
    const sortCfg = tableSorts.scanEntries || { field: 'rel_path', dir: 'asc' };
    return [...list].sort((a, b) => {
      switch (sortCfg.field) {
        case 'is_dir':
          return compareValues(a.is_dir ? 'DIR' : 'FILE', b.is_dir ? 'DIR' : 'FILE', sortCfg.dir);
        case 'hash':
          return compareValues(a.hash || '', b.hash || '', sortCfg.dir);
        case 'size':
          return compareValues(a.size || 0, b.size || 0, sortCfg.dir);
        case 'mode':
          return compareValues(a.mode || '', b.mode || '', sortCfg.dir);
        case 'owner':
          return compareValues(`${a.owner || ''}:${a.group || ''}`, `${b.owner || ''}:${b.group || ''}`, sortCfg.dir);
        case 'rel_path':
        default:
          return compareValues(a.rel_path || '', b.rel_path || '', sortCfg.dir);
      }
    });
  }, [selectedScan, entryFilter, scanEntryKindFilter, tableSorts.scanEntries]);

  const filteredDuplicates = useMemo(() => {
    const q = entryFilter.trim().toLowerCase();
    const list = !q
      ? [...duplicates]
      : duplicates.filter(
          (group) =>
            group.hash?.toLowerCase().includes(q) ||
            (group.files || []).some(
              (f) =>
                f.name?.toLowerCase().includes(q) ||
                f.abs_path?.toLowerCase().includes(q) ||
                f.rel_path?.toLowerCase().includes(q) ||
                f.hostname?.toLowerCase().includes(q)
            )
        );
    const sortCfg = tableSorts.dups || { field: 'wasted_bytes', dir: 'desc' };
    return list.sort((a, b) => {
      switch (sortCfg.field) {
        case 'hash':
          return compareValues(a.hash || '', b.hash || '', sortCfg.dir);
        case 'size':
          return compareValues(a.size || 0, b.size || 0, sortCfg.dir);
        case 'count':
          return compareValues(a.count || 0, b.count || 0, sortCfg.dir);
        case 'location':
          return compareValues(
            `${a.files?.[0]?.hostname || ''}:${a.files?.[0]?.abs_path || ''}`,
            `${b.files?.[0]?.hostname || ''}:${b.files?.[0]?.abs_path || ''}`,
            sortCfg.dir
          );
        case 'wasted_bytes':
        default:
          return compareValues(a.wasted_bytes || 0, b.wasted_bytes || 0, sortCfg.dir);
      }
    });
  }, [duplicates, entryFilter, tableSorts.dups]);

  const sortedDashboardDuplicates = useMemo(() => {
    const sortCfg = tableSorts.dashDups || { field: 'wasted_bytes', dir: 'desc' };
    return [...filteredDuplicates].sort((a, b) => {
      switch (sortCfg.field) {
        case 'name':
          return compareValues(
            a.files?.[0]?.name || a.hash || '',
            b.files?.[0]?.name || b.hash || '',
            sortCfg.dir
          );
        case 'count':
          return compareValues(a.count || 0, b.count || 0, sortCfg.dir);
        case 'wasted_bytes':
        default:
          return compareValues(a.wasted_bytes || 0, b.wasted_bytes || 0, sortCfg.dir);
      }
    });
  }, [filteredDuplicates, tableSorts.dashDups]);

  const filteredDiffChanges = useMemo(() => {
    const changes = diffReport?.changes || [];
    const q = entryFilter.trim().toLowerCase();
    const list = changes.filter((c) => {
      if (diffChangeFilter !== 'all' && c.change !== diffChangeFilter) return false;
      if (!q) return true;
      return (
        c.rel_path?.toLowerCase().includes(q) ||
        c.base_hash?.toLowerCase().includes(q) ||
        c.target_hash?.toLowerCase().includes(q)
      );
    });
    const sortCfg = tableSorts.diff || { field: 'rel_path', dir: 'asc' };
    return [...list].sort((a, b) => {
      switch (sortCfg.field) {
        case 'change':
          return compareValues(a.change || '', b.change || '', sortCfg.dir);
        case 'is_dir':
          return compareValues(a.is_dir ? 'DIR' : 'FILE', b.is_dir ? 'DIR' : 'FILE', sortCfg.dir);
        case 'base_hash':
          return compareValues(a.base_hash || '', b.base_hash || '', sortCfg.dir);
        case 'target_hash':
          return compareValues(a.target_hash || '', b.target_hash || '', sortCfg.dir);
        case 'size':
          return compareValues(
            a.target_size ?? a.base_size ?? 0,
            b.target_size ?? b.base_size ?? 0,
            sortCfg.dir
          );
        case 'rel_path':
        default:
          return compareValues(a.rel_path || '', b.rel_path || '', sortCfg.dir);
      }
    });
  }, [diffReport, entryFilter, diffChangeFilter, tableSorts.diff]);

  const renderOpCard = ({
    id,
    cardKey,
    selected,
    badgeTone = 'dark',
    title,
    versionBadge,
    pct = 0,
    leftTitle,
    leftStatus = 'Synced',
    leftStatusTone = 'ok',
    rightTitle,
    rightDelta,
    pills = [],
    shareBadge,
    onClick,
    onDoubleClick,
  }) => {
    const clampedPct = Math.max(4, Math.min(100, Math.round(pct || 0)));
    const radius = 8.5;
    const circum = 2 * Math.PI * radius;
    const dashOffset = circum * (1 - clampedPct / 100);
    const ringColor =
      badgeTone === 'red' ? '#e5484d' : selected ? '#3b63eb' : '#d9532f';

    return (
      <div
        key={cardKey}
        id={id}
        role="button"
        tabIndex={0}
        aria-pressed={Boolean(selected)}
        className={`op-card ${selected ? 'selected' : ''}`}
        onClick={onClick}
        onDoubleClick={onDoubleClick}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            if (onClick) onClick(e);
          }
        }}
      >
        <div className="op-card-top">
          <div className="op-card-title-group">
            <span className={`op-card-icon-badge ${badgeTone}`} aria-hidden="true">
              {badgeTone === 'red' ? '⚡' : '▣'}
            </span>
            <span className="op-card-title" title={title}>
              {title}
            </span>
            {versionBadge && (
              <span className="op-card-version">{versionBadge}</span>
            )}
          </div>
          <svg className="op-card-ring" width="22" height="22" viewBox="0 0 22 22" aria-hidden="true">
            <circle
              cx="11"
              cy="11"
              r={radius}
              fill="none"
              stroke={darkMode ? '#283548' : '#e6e8ec'}
              strokeWidth="2.2"
            />
            <circle
              cx="11"
              cy="11"
              r={radius}
              fill="none"
              stroke={ringColor}
              strokeWidth="2.2"
              strokeDasharray={circum.toFixed(1)}
              strokeDashoffset={dashOffset.toFixed(1)}
              strokeLinecap="round"
              transform="rotate(-90 11 11)"
            />
          </svg>
        </div>

        <div className="op-card-route">
          <div className="op-route-side">
            <div className="op-route-code">{leftTitle}</div>
            <div className={`op-route-status ${leftStatusTone}`}>{leftStatus}</div>
          </div>

          <div className="op-route-connector" aria-hidden="true">
            <span className="op-route-line" />
            <span className="op-route-dot-small" />
            <span className="op-route-node" />
          </div>

          <div className="op-route-side right">
            <div className="op-route-code">{rightTitle}</div>
            <div className="op-route-delta">{rightDelta}</div>
          </div>
        </div>

        <div className="op-card-footer">
          <div className="op-pill-group">
            {pills.map((p, idx) => (
              <span key={idx} className="op-meta-pill">
                {p}
              </span>
            ))}
          </div>
          {shareBadge && (
            <span className="op-share-pill">{shareBadge}</span>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className={`app-shell ${darkMode ? 'theme-dark' : ''}`}>
      <aside
        id="primary-sidebar"
        className={`sidebar-rail ${sidebarExpanded ? 'expanded' : ''}`}
        aria-label="Primary Sidebar Navigation"
      >
        <div className="sidebar-top">
          <div className="sidebar-brand-row">
            <button
              id="brand-expand-btn"
              type="button"
              className="brand-mark-btn"
              onClick={() => setSidebarExpanded((prev) => !prev)}
              title={sidebarExpanded ? 'Collapse sidebar' : 'Expand sidebar'}
              aria-expanded={sidebarExpanded}
            >
              <span className="brand-glyph" aria-hidden="true">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                  <path
                    d="M12 4C7.58172 4 4 7.58172 4 12C4 16.4183 7.58172 20 12 20H19C19.5523 20 20 19.5523 20 19V12C20 7.58172 16.4183 4 12 4Z"
                    stroke="currentColor"
                    strokeWidth="2.4"
                  />
                  <circle cx="12" cy="12" r="2.2" fill="currentColor" />
                </svg>
              </span>
              <span className="brand-text-wrap">
                <span className="brand-title">go-ls</span>
                <span className="brand-subtitle">Merkle Snapshot Console</span>
              </span>
            </button>
          </div>

          <nav className="sidebar-nav" aria-label="Main Views">
            <button
              id="tab-btn-dashboard"
              type="button"
              className={`rail-item ${activeTab === 'dashboard' ? 'active' : ''}`}
              onClick={() => setActiveTab('dashboard')}
              title="Overview Dashboard"
            >
              <span className="rail-icon" aria-hidden="true">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3.5" y="3.5" width="7.5" height="7.5" rx="2" />
                  <rect x="13" y="3.5" width="7.5" height="7.5" rx="2" />
                  <rect x="3.5" y="13" width="7.5" height="7.5" rx="2" />
                  <rect x="13" y="13" width="7.5" height="7.5" rx="2" />
                </svg>
              </span>
              <span className="rail-label">Dashboard</span>
            </button>

            <button
              id="tab-btn-scans"
              type="button"
              className={`rail-item ${activeTab === 'scans' ? 'active' : ''}`}
              onClick={() => setActiveTab('scans')}
              title="Computers & Directory Explorer"
            >
              <span className="rail-icon" aria-hidden="true">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="4" y="5" width="16" height="14" rx="4" />
                  <circle cx="9.5" cy="12" r="1.4" fill="currentColor" />
                  <circle cx="14.5" cy="12" r="1.4" fill="currentColor" />
                </svg>
              </span>
              <span className="rail-label">Computers &amp; Scans</span>
              <span className="rail-count">{scans.length}</span>
            </button>

            <button
              id="tab-btn-tree"
              type="button"
              className={`rail-item ${activeTab === 'tree' ? 'active' : ''}`}
              onClick={() => setActiveTab('tree')}
              title="WinDirStat Directory Tree Explorer"
            >
              <span className="rail-icon" aria-hidden="true">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M4 5H10V9H4V5Z" />
                  <path d="M7 9V18H12" />
                  <path d="M7 13H12" />
                  <rect x="12" y="11" width="8" height="4" rx="1" />
                  <rect x="12" y="16.5" width="8" height="4" rx="1" />
                </svg>
              </span>
              <span className="rail-label">Directory Tree</span>
            </button>

            <button
              id="tab-btn-treemap"
              type="button"
              className={`rail-item ${activeTab === 'treemap' ? 'active' : ''}`}
              onClick={() => setActiveTab('treemap')}
              title="WinDirStat 2D Cushion Treemap"
            >
              <span className="rail-icon" aria-hidden="true">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3.5" y="3.5" width="17" height="7" rx="2" />
                  <rect x="3.5" y="13.5" width="9.5" height="7" rx="2" />
                  <rect x="15.5" y="13.5" width="5" height="7" rx="1.5" />
                </svg>
              </span>
              <span className="rail-label">Cushion Treemap</span>
            </button>

            <button
              id="tab-btn-filetypes"
              type="button"
              className={`rail-item ${activeTab === 'filetypes' ? 'active' : ''}`}
              onClick={() => setActiveTab('filetypes')}
              title="File Types Distribution Analyzer"
            >
              <span className="rail-icon" aria-hidden="true">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3.5" y="5" width="17" height="5" rx="1.5" />
                  <path d="M9 5V10" />
                  <path d="M15 5V10" />
                  <path d="M4 14.5H16" />
                  <path d="M4 18.5H12" />
                </svg>
              </span>
              <span className="rail-label">File Types</span>
            </button>

            <button
              id="tab-btn-diff"
              type="button"
              className={`rail-item ${activeTab === 'diff' ? 'active' : ''}`}
              onClick={() => setActiveTab('diff')}
              title="Compare / Diff Snapshots"
            >
              <span className="rail-icon" aria-hidden="true">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3.5" y="3.5" width="17" height="17" rx="4" />
                  <path d="M8.5 15.5V10.5" />
                  <path d="M12 15.5V8" />
                  <path d="M15.5 15.5V12" />
                </svg>
              </span>
              <span className="rail-label">Compare / Diff</span>
            </button>

            <button
              id="tab-btn-duplicates"
              type="button"
              className={`rail-item ${activeTab === 'duplicates' ? 'active' : ''}`}
              onClick={() => setActiveTab('duplicates')}
              title="Duplicates Analyzer"
            >
              <span className="rail-icon" aria-hidden="true">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                  <circle cx="8.5" cy="8.5" r="3.5" />
                  <circle cx="15.5" cy="8.5" r="3.5" />
                  <circle cx="8.5" cy="15.5" r="3.5" />
                  <circle cx="15.5" cy="15.5" r="3.5" />
                </svg>
              </span>
              <span className="rail-label">Duplicates</span>
            </button>
          </nav>
        </div>

        <div className="sidebar-bottom">
          <button
            id="rail-theme-btn"
            type="button"
            className="rail-item"
            onClick={handleToggleTheme}
            aria-pressed={darkMode}
            title={darkMode ? 'Switch to Light Theme' : 'Switch to Dark Theme'}
          >
            <span className="rail-icon" aria-hidden="true">
              {darkMode ? (
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="4.5" />
                  <path d="M12 2.5V5M12 19V21.5M4.93 4.93L6.7 6.7M17.3 17.3L19.07 19.07M2.5 12H5M19 12H21.5M4.93 19.07L6.7 17.3M17.3 6.7L19.07 4.93" />
                </svg>
              ) : (
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79Z" />
                </svg>
              )}
            </span>
            <span className="rail-label">
              {darkMode ? 'Light Theme' : 'Dark Theme'}
            </span>
          </button>

          <button
            id="rail-refresh-btn"
            type="button"
            className="rail-item"
            onClick={fetchScans}
            disabled={loadingScans}
            title="Refresh snapshots from server"
          >
            <span className="rail-icon" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
                <rect x="3.5" y="3.5" width="17" height="17" rx="4" />
                <path d="M7.5 13.5L10.5 10.5L13 13L16.5 9.5" />
              </svg>
            </span>
            <span className="rail-label">
              {loadingScans ? 'Syncing...' : 'Sync Server'}
            </span>
          </button>

          <button
            id="sidebar-toggle-btn"
            type="button"
            className="rail-item"
            onClick={() => setSidebarExpanded((prev) => !prev)}
            title={sidebarExpanded ? 'Collapse sidebar' : 'Expand sidebar'}
            aria-expanded={sidebarExpanded}
          >
            <span className="rail-icon" aria-hidden="true">
              {sidebarExpanded ? (
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.1" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="11 17 6 12 11 7" />
                  <polyline points="18 17 13 12 18 7" />
                </svg>
              ) : (
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.1" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="6 17 11 12 6 7" />
                  <polyline points="13 17 18 12 13 7" />
                </svg>
              )}
            </span>
            <span className="rail-label">
              {sidebarExpanded ? 'Collapse Sidebar' : 'Expand Sidebar'}
            </span>
          </button>

          <button
            id="rail-storage-btn"
            type="button"
            className="rail-item"
            onClick={() => setActiveTab('scans')}
            title={`Indexed Volume: ${formatBytes(totalBytesAcrossScans)}`}
          >
            <span className="rail-icon" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <rect x="4" y="6" width="16" height="13" rx="3" />
                <path d="M8 6V5C8 3.89543 8.89543 3 10 3H14C15.1046 3 16 3.89543 16 5V6" />
                <circle cx="12" cy="12.5" r="1.3" fill="currentColor" />
              </svg>
            </span>
            <span className="rail-label">{formatBytes(totalBytesAcrossScans)}</span>
          </button>

          <div
            className="rail-item"
            id="server-health-badge"
            title={serverOnline ? 'go-ls API Server Online' : 'go-ls API Server Offline'}
          >
            <span className="rail-icon" aria-hidden="true">
              <span className={`status-orb ${serverOnline ? 'online' : 'offline'}`} />
            </span>
            <span className="rail-label">{serverOnline ? 'API Online' : 'API Offline'}</span>
          </div>
        </div>
      </aside>

      <div className="main-viewport">
        <main className="main-content">
          <header className="viewport-header">
            <div className="viewport-brand-crumbs">
              <span className="viewport-crumb-glyph" aria-hidden="true">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M4 19L10 5L15 15L20 11" />
                </svg>
              </span>
              <h1 className="viewport-title">
                {activeTab === 'dashboard'
                  ? 'Operations Overview'
                  : activeTab === 'tree'
                    ? 'Directory Tree'
                    : activeTab === 'treemap'
                      ? '3D Cushion Map'
                      : activeTab === 'filetypes'
                        ? 'File Types'
                        : activeTab === 'scans'
                          ? 'Computers & Scans'
                          : activeTab === 'diff'
                            ? 'Snapshot Diff'
                            : 'Duplicates'}
              </h1>

              {(activeTab === 'tree' || activeTab === 'treemap') && treeBreadcrumbs.length > 0 && (
                <nav className="viewport-crumb-trail" aria-label="Focused Subtree Breadcrumbs">
                  {treeBreadcrumbs.map((crumb, idx) => {
                    const isLast = idx === treeBreadcrumbs.length - 1;
                    return (
                      <span key={crumb.key} className="viewport-crumb-step">
                        <span className="viewport-crumb-sep" aria-hidden="true">/</span>
                        <button
                          type="button"
                          className={`viewport-crumb-btn ${isLast ? 'current' : ''}`}
                          onClick={() => {
                            setFocusedTreeKey(crumb.key);
                            setSelectedTreemapFile(null);
                          }}
                          title={`Focus ${crumb.name} (${formatBytes(crumb.subtreeBytes)})`}
                        >
                          {crumb.name}
                        </button>
                      </span>
                    );
                  })}
                </nav>
              )}

              {activeTab === 'scans' && selectedScan && (
                <div className="viewport-crumb-trail" aria-label="Selected Snapshot Breadcrumb">
                  <span className="viewport-crumb-step">
                    <span className="viewport-crumb-sep" aria-hidden="true">/</span>
                    <span className="viewport-crumb-btn current">
                      {selectedScan.hostname}:{selectedScan.root_path}
                    </span>
                  </span>
                </div>
              )}
            </div>

            <div className="viewport-search-pill">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <circle cx="11" cy="11" r="7.5" />
                <line x1="16.5" y1="16.5" x2="21" y2="21" />
              </svg>
              <input
                ref={searchInputRef}
                id="global-search-input"
                type="search"
                className="viewport-search-input"
                placeholder="Search computers, directories, file paths, or Merkle hashes..."
                value={entryFilter}
                onChange={(e) => setEntryFilter(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    setEntryFilter('');
                    e.currentTarget.blur();
                  }
                }}
              />
              {entryFilter ? (
                <button
                  id="global-search-clear-btn"
                  type="button"
                  className="search-clear-btn"
                  onClick={() => {
                    setEntryFilter('');
                    searchInputRef.current?.focus();
                  }}
                  title="Clear search filter (Esc)"
                  aria-label="Clear search filter"
                >
                  ✕
                </button>
              ) : (
                <kbd className="search-kbd-hint" aria-hidden="true">/</kbd>
              )}
            </div>

            <div className="toolbar-group">
              {activeTab === 'tree' && (
                <div className="op-top-option-bar" role="toolbar" aria-label="Directory Tree Options">
                  <button
                    id="windir-expand-all-btn"
                    type="button"
                    className="op-floor-tab"
                    onClick={handleExpandAllTree}
                  >
                    <span>Expand All</span>
                  </button>
                  <button
                    id="windir-collapse-btn"
                    type="button"
                    className="op-floor-tab"
                    onClick={handleCollapseTreeToRoots}
                  >
                    <span>Collapse</span>
                  </button>
                  {focusedTreeNode?.parentKey && (
                    <button
                      id="windir-up-level-btn"
                      type="button"
                      className="op-floor-tab"
                      onClick={() => {
                        setFocusedTreeKey(focusedTreeNode.parentKey);
                        setSelectedTreemapFile(null);
                      }}
                    >
                      <span>↑ Up Level</span>
                    </button>
                  )}
                  {focusedTreeNode && winDirRoot && focusedTreeNode.key !== winDirRoot.key && (
                    <button
                      id="windir-reset-focus-btn"
                      type="button"
                      className="op-floor-tab"
                      onClick={() => {
                        setFocusedTreeKey(winDirRoot.key);
                        setSelectedTreemapFile(null);
                      }}
                    >
                      <span>Reset Root</span>
                    </button>
                  )}
                  <button
                    id="tree-open-treemap-btn"
                    type="button"
                    className="op-floor-tab active"
                    onClick={() => setActiveTab('treemap')}
                  >
                    <span className="op-floor-dot" aria-hidden="true" />
                    <span>3D Cushion Map →</span>
                  </button>
                </div>
              )}

              {activeTab === 'treemap' && (
                <>
                  <div className="op-top-option-bar" role="toolbar" aria-label="3D Cushion Map Options">
                    <button
                      id="treemap-mode-modules-btn"
                      type="button"
                      className={`op-floor-tab ${treemapMode === 'modules' ? 'active' : ''}`}
                      onClick={() => {
                        setTreemapMode('modules');
                        setSelectedTreemapFile(null);
                      }}
                    >
                      {treemapMode === 'modules' && <span className="op-floor-dot" aria-hidden="true" />}
                      <span>3D Subtree Modules</span>
                    </button>
                    <button
                      id="treemap-mode-files-btn"
                      type="button"
                      className={`op-floor-tab ${treemapMode === 'files' ? 'active' : ''}`}
                      onClick={() => {
                        setTreemapMode('files');
                        setSelectedTreemapFile(null);
                      }}
                    >
                      {treemapMode === 'files' && <span className="op-floor-dot" aria-hidden="true" />}
                      <span>3D Leaf Files</span>
                    </button>
                    <button
                      id="treemap-toggle-legend-btn"
                      type="button"
                      className={`op-floor-tab ${showTreemapLegend ? 'active' : ''}`}
                      onClick={() => setShowTreemapLegend((prev) => !prev)}
                    >
                      {showTreemapLegend && <span className="op-floor-dot" aria-hidden="true" />}
                      <span>Legend ({extensionStats.length})</span>
                    </button>
                    {focusedTreeNode?.parentKey && (
                      <button
                        id="treemap-up-level-btn"
                        type="button"
                        className="op-floor-tab"
                        onClick={() => {
                          setFocusedTreeKey(focusedTreeNode.parentKey);
                          setSelectedTreemapFile(null);
                        }}
                      >
                        <span>↑ Up Level</span>
                      </button>
                    )}
                    {focusedTreeNode && winDirRoot && focusedTreeNode.key !== winDirRoot.key && (
                      <button
                        id="treemap-reset-focus-btn"
                        type="button"
                        className="op-floor-tab"
                        onClick={() => {
                          setFocusedTreeKey(winDirRoot.key);
                          setSelectedTreemapFile(null);
                        }}
                      >
                        <span>Reset Root</span>
                      </button>
                    )}
                    <button
                      id="treemap-back-to-tree-btn"
                      type="button"
                      className="op-floor-tab"
                      onClick={() => setActiveTab('tree')}
                    >
                      <span>← Directory Tree</span>
                    </button>
                  </div>

                  {selectedTreemapFile && (
                    <div className="toolbar-group">
                      {selectedTreemapFile.isDir && (
                        <button
                          id="treemap-drill-dir-btn"
                          type="button"
                          className="btn btn-primary"
                          onClick={() => {
                            setFocusedTreeKey(selectedTreemapFile.key);
                            setSelectedTreemapFile(null);
                          }}
                        >
                          Drill into Folder →
                        </button>
                      )}
                      {selectedTreemapFile.scanId && (
                        <button
                          id="treemap-open-scan-btn"
                          type="button"
                          className="btn"
                          onClick={() => {
                            setSelectedScanId(selectedTreemapFile.scanId);
                            setEntryFilter(selectedTreemapFile.isDir ? '' : selectedTreemapFile.name || '');
                            setActiveTab('scans');
                          }}
                        >
                          Locate in Snapshot →
                        </button>
                      )}
                    </div>
                  )}
                </>
              )}

              {activeTab === 'filetypes' && (
                <div className="op-top-option-bar" role="toolbar" aria-label="File Type Metric Mode">
                  <button
                    id="filetypes-metric-bytes-btn"
                    type="button"
                    className={`op-floor-tab ${fileTypeMetric === 'bytes' ? 'active' : ''}`}
                    onClick={() => setFileTypeMetric('bytes')}
                  >
                    {fileTypeMetric === 'bytes' && <span className="op-floor-dot" aria-hidden="true" />}
                    <span>Weight by Bytes</span>
                  </button>
                  <button
                    id="filetypes-metric-count-btn"
                    type="button"
                    className={`op-floor-tab ${fileTypeMetric === 'count' ? 'active' : ''}`}
                    onClick={() => setFileTypeMetric('count')}
                  >
                    {fileTypeMetric === 'count' && <span className="op-floor-dot" aria-hidden="true" />}
                    <span>Weight by File Count</span>
                  </button>
                  {selectedFileTypeExt && (
                    <button
                      id="filetypes-clear-ext-btn"
                      type="button"
                      className="op-floor-tab"
                      onClick={() => setSelectedFileTypeExt('')}
                    >
                      <span>Clear ({selectedFileTypeExt})</span>
                    </button>
                  )}
                </div>
              )}

              {activeTab === 'scans' && selectedScan && (
                <div className="op-top-option-bar" role="toolbar" aria-label="Snapshot View Mode">
                  <button
                    id="view-entries-btn"
                    type="button"
                    className={`op-floor-tab ${detailView === 'entries' ? 'active' : ''}`}
                    onClick={() => setDetailView('entries')}
                  >
                    {detailView === 'entries' && <span className="op-floor-dot" aria-hidden="true" />}
                    <span>Current Files ({filteredEntries.length})</span>
                  </button>
                  <button
                    id="view-history-btn"
                    type="button"
                    className={`op-floor-tab ${detailView === 'history' ? 'active' : ''}`}
                    onClick={() => setDetailView('history')}
                  >
                    {detailView === 'history' && <span className="op-floor-dot" aria-hidden="true" />}
                    <span>Change History ({(selectedScan.history || []).length})</span>
                  </button>
                </div>
              )}

              {activeTab === 'diff' && diffReport && (
                <div className="op-top-option-bar" role="toolbar" aria-label="Diff Filter Options">
                  <button
                    type="button"
                    className={`op-floor-tab ${diffChangeFilter === 'all' ? 'active' : ''}`}
                    onClick={() => setDiffChangeFilter('all')}
                  >
                    {diffChangeFilter === 'all' && <span className="op-floor-dot" aria-hidden="true" />}
                    <span>All ({(diffReport.changes || []).length})</span>
                  </button>
                  <button
                    type="button"
                    className={`op-floor-tab ${diffChangeFilter === 'added' ? 'active' : ''}`}
                    onClick={() => setDiffChangeFilter((prev) => (prev === 'added' ? 'all' : 'added'))}
                  >
                    <span className="diff-badge added">+{diffReport.added} Added</span>
                  </button>
                  <button
                    type="button"
                    className={`op-floor-tab ${diffChangeFilter === 'modified' ? 'active' : ''}`}
                    onClick={() => setDiffChangeFilter((prev) => (prev === 'modified' ? 'all' : 'modified'))}
                  >
                    <span className="diff-badge modified">~{diffReport.modified} Modified</span>
                  </button>
                  <button
                    type="button"
                    className={`op-floor-tab ${diffChangeFilter === 'deleted' ? 'active' : ''}`}
                    onClick={() => setDiffChangeFilter((prev) => (prev === 'deleted' ? 'all' : 'deleted'))}
                  >
                    <span className="diff-badge deleted">-{diffReport.deleted} Deleted</span>
                  </button>
                  <span className="tag-badge" style={{ marginInline: '0.35rem' }}>={diffReport.unchanged} Unchanged</span>
                </div>
              )}

              {activeTab === 'duplicates' && (
                <div className="op-top-option-bar" role="toolbar" aria-label="Duplicates Actions">
                  <span className="op-floor-tab" style={{ cursor: 'default' }}>
                    Wasted: {formatBytes(totalWastedBytes)}
                  </span>
                  <button
                    id="refresh-duplicates-btn"
                    type="button"
                    className="op-floor-tab active"
                    onClick={() => fetchDuplicates(dupScanFilter)}
                    disabled={loadingDups}
                  >
                    <span className="op-floor-dot" aria-hidden="true" />
                    <span>{loadingDups ? 'Refreshing...' : 'Refresh Duplicates'}</span>
                  </button>
                </div>
              )}

              <button
                id="header-theme-toggle-btn"
                type="button"
                className="op-icon-btn"
                onClick={handleToggleTheme}
                aria-pressed={darkMode}
                title={darkMode ? 'Switch to Light Theme' : 'Switch to Dark Theme'}
                aria-label={darkMode ? 'Switch to Light Theme' : 'Switch to Dark Theme'}
              >
                {darkMode ? (
                  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <circle cx="12" cy="12" r="4.5" />
                    <path d="M12 2.5V5M12 19V21.5M4.93 4.93L6.7 6.7M17.3 17.3L19.07 19.07M2.5 12H5M19 12H21.5M4.93 19.07L6.7 17.3M17.3 6.7L19.07 4.93" />
                  </svg>
                ) : (
                  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79Z" />
                  </svg>
                )}
              </button>

              {activeTab !== 'dashboard' && (
                <button
                  id="header-drawer-toggle-btn"
                  type="button"
                  className="op-icon-btn"
                  onClick={() => setOpDrawerOpen((prev) => !prev)}
                  title={opDrawerOpen ? 'Hide options panel' : 'Show options panel'}
                >
                  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <rect x="3" y="4" width="18" height="16" rx="2.5" />
                    <line x1="9.5" y1="4" x2="9.5" y2="20" />
                  </svg>
                </button>
              )}
            </div>
          </header>

          {errorMsg && (
            <div className="error-banner" role="alert" id="error-alert">
              {errorMsg}
            </div>
          )}

        {activeTab === 'dashboard' && (
          <div className="dashboard-viewport-body" id="dashboard-workspace">
            <section className="metrics-strip" aria-label="Selected Computer Overview Metrics">
              <div
                className="metric-box interactive"
                role="button"
                tabIndex={0}
                onClick={() => setActiveTab('scans')}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    setActiveTab('scans');
                  }
                }}
              >
                <div className="metric-label">Selected Computer</div>
                <div className="metric-value">{activeComputer?.hostname || '—'}</div>
                <div className="metric-subtext">
                  {activeComputerDrivesCount} drive(s) · {effectiveComputers.length} host(s) · Inspect →
                </div>
                <div className="metric-bar-track" aria-hidden="true">
                  <div className="metric-bar-fill" style={{ width: `${activeComputer ? 100 : 8}%` }} />
                </div>
              </div>
              <div
                className="metric-box interactive"
                role="button"
                tabIndex={0}
                onClick={() => setActiveTab('tree')}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    setActiveTab('tree');
                  }
                }}
              >
                <div className="metric-label">Tracked Directories</div>
                <div className="metric-value">{activeComputerScans.length}</div>
                <div className="metric-subtext">Active Merkle snapshots · Tree →</div>
                <div className="metric-bar-track" aria-hidden="true">
                  <div className="metric-bar-fill" style={{ width: `${activeComputerScans.length > 0 ? 100 : 8}%` }} />
                </div>
              </div>
              <div
                className="metric-box interactive"
                role="button"
                tabIndex={0}
                onClick={() => {
                  if (activeComputer) setFileTypeScope(`comp:${activeComputer.id}`);
                  setActiveTab('filetypes');
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    if (activeComputer) setFileTypeScope(`comp:${activeComputer.id}`);
                    setActiveTab('filetypes');
                  }
                }}
              >
                <div className="metric-label">Indexed Files</div>
                <div className="metric-value">{activeComputerFiles.toLocaleString()}</div>
                <div className="metric-subtext">{activeComputerDirs.toLocaleString()} subdirectories · File Types →</div>
                <div className="metric-bar-track" aria-hidden="true">
                  <div className="metric-bar-fill" style={{ width: `${activeComputerFiles > 0 ? 100 : 8}%` }} />
                </div>
              </div>
              <div
                className="metric-box interactive"
                role="button"
                tabIndex={0}
                onClick={() => setActiveTab('treemap')}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    setActiveTab('treemap');
                  }
                }}
              >
                <div className="metric-label">Indexed Volume</div>
                <div className="metric-value">{formatBytes(activeComputerBytes)}</div>
                <div className="metric-subtext">On {activeComputer?.hostname || 'selected host'} · 3D Map →</div>
                <div className="metric-bar-track" aria-hidden="true">
                  <div className="metric-bar-fill" style={{ width: `${activeComputerBytes > 0 ? 100 : 8}%` }} />
                </div>
              </div>
              <div
                className="metric-box interactive"
                role="button"
                tabIndex={0}
                onClick={() => setActiveTab('duplicates')}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    setActiveTab('duplicates');
                  }
                }}
              >
                <div className="metric-label">Duplicate Wasted Space</div>
                <div className="metric-value">{formatBytes(activeComputerWastedBytes)}</div>
                <div className="metric-subtext">{activeComputerDuplicates.length} duplicate group(s) · Deduplicate →</div>
                <div className="metric-bar-track" aria-hidden="true">
                  <div
                    className="metric-bar-fill warn"
                    style={{
                      width: `${
                        activeComputerBytes > 0
                          ? Math.min(100, Math.max(8, Math.round((activeComputerWastedBytes / activeComputerBytes) * 100)))
                          : 8
                      }%`,
                    }}
                  />
                </div>
              </div>
            </section>

            <div className="dashboard-grid" id="dashboard-overview">
            <div className="dashboard-stack">
              <section className="hexbin-update-card" id="dashboard-hexbin-card" aria-label="Directory Update Hexbin Chart">
                <div className="hexbin-card-header">
                  <div className="hexbin-period-tabs" role="tablist" aria-label="Hexbin Activity Period">
                    {[
                      { id: 'day', label: 'Day' },
                      { id: 'week', label: 'Week' },
                      { id: 'month', label: 'Month' },
                      { id: 'year', label: 'Last year' },
                    ].map((tab) => {
                      const isCurrent = hexbinPeriod === tab.id;
                      return (
                        <button
                          key={tab.id}
                          id={`hexbin-period-${tab.id}`}
                          type="button"
                          role="tab"
                          aria-selected={isCurrent}
                          className={`hexbin-period-tab ${isCurrent ? 'active' : ''}`}
                          onClick={() => setHexbinPeriod(tab.id)}
                        >
                          {tab.label}
                        </button>
                      );
                    })}
                  </div>
                  <div className="hexbin-header-controls">
                    <label htmlFor="hexbin-computer-select" className="hexbin-select-wrap">
                      <span className="hexbin-select-label">Computer</span>
                      <select
                        id="hexbin-computer-select"
                        className="hexbin-computer-select"
                        value={activeComputer?.id || ''}
                        onChange={(e) => setSelectedComputerId(e.target.value)}
                        aria-label="Select computer for dashboard"
                      >
                        {effectiveComputers.map((comp) => (
                          <option key={comp.id} value={comp.id}>
                            {comp.hostname} ({(comp.directories || []).length} dir{(comp.directories || []).length === 1 ? '' : 's'})
                          </option>
                        ))}
                      </select>
                    </label>
                    <span
                      className="hexbin-op-badge"
                      style={{
                        backgroundColor: HEXBIN_OP_PALETTES.create.pillBg,
                        color: HEXBIN_OP_PALETTES.create.pillText,
                      }}
                      title={`Created entries: +${activeComputerOpSummary.createCount}`}
                    >
                      CREATE +{activeComputerOpSummary.createCount}
                    </span>
                    <span
                      className="hexbin-op-badge"
                      style={{
                        backgroundColor: HEXBIN_OP_PALETTES.update.pillBg,
                        color: HEXBIN_OP_PALETTES.update.pillText,
                      }}
                      title={`Updated entries: ~${activeComputerOpSummary.updateCount}`}
                    >
                      UPDATE ~{activeComputerOpSummary.updateCount}
                    </span>
                    <span
                      className="hexbin-op-badge"
                      style={{
                        backgroundColor: HEXBIN_OP_PALETTES.delete.pillBg,
                        color: HEXBIN_OP_PALETTES.delete.pillText,
                      }}
                      title={`Deleted entries: -${activeComputerOpSummary.deleteCount}`}
                    >
                      DELETE -{activeComputerOpSummary.deleteCount}
                    </span>
                    {scans.length > 1 && (
                      <button
                        id="dash-open-diff-btn"
                        type="button"
                        className="hexbin-action-btn"
                        onClick={() => setActiveTab('diff')}
                      >
                        Compare Snapshots →
                      </button>
                    )}
                  </div>
                </div>

                {hexbinVisibleScans.length === 0 ? (
                  <div className="empty-state">
                    No directory snapshots match the current view.
                  </div>
                ) : (
                  <div className="hexbin-body-scroll">
                    <div className="hexbin-time-axis" aria-hidden="true">
                      <span className="hexbin-axis-spacer" />
                      <div className="hexbin-axis-ticks">
                        {(HEXBIN_PERIOD_AXIS[hexbinPeriod] || HEXBIN_PERIOD_AXIS.day).map((label, idx) => (
                          <span key={`${label}-${idx}`} className="hexbin-axis-label">
                            {label}
                          </span>
                        ))}
                      </div>
                    </div>

                    <div className="hexbin-bands-list" ref={hexbinBandsRef}>
                      {(() => {
                        const primaryScan = hexbinVisibleScans[0];
                        const palette = activeComputerOpSummary.palette;
                        const availW = Math.max(320, hexbinBox.width - 44);
                        const availH = Math.max(220, hexbinBox.height);
                        const R = 15.5;
                        const W = Math.sqrt(3) * R;
                        const dx = 30.2;
                        const dy = 26.2;
                        const padY = 17.5;
                        const rows = Math.max(10, Math.round((availH - padY * 2) / dy) + 1);
                        const svgHeight = Math.ceil(padY * 2 + (rows - 1) * dy);
                        const targetSvgWidth = Math.max(360, svgHeight * (availW / availH));
                        const cols = Math.max(
                          14,
                          Math.floor((targetSvgWidth - (W + 4) - dx * 0.5) / dx) + 1
                        );
                        const gridSpanW = (cols - 1) * dx + dx * 0.5;
                        const padX = Math.max(W / 2 + 2, (targetSvgWidth - gridSpanW) / 2);
                        const svgWidth = Math.ceil(padX * 2 + gridSpanW);
                        const seedKey = [
                          activeComputer?.id || '',
                          hexbinPeriod,
                          activeComputerOpSummary.predominantOp,
                          ...hexbinVisibleScans.map((s) => `${s.id}:${s.root_hash || ''}:${s.version || 1}`),
                        ].join('|');
                        const cells = buildHexbinBandCells(seedKey, rows, cols, activeComputerOpSummary);
                        const hexPoints = (cx, cy) =>
                          [
                            `${cx.toFixed(1)},${(cy - R).toFixed(1)}`,
                            `${(cx + W / 2).toFixed(1)},${(cy - R / 2).toFixed(1)}`,
                            `${(cx + W / 2).toFixed(1)},${(cy + R / 2).toFixed(1)}`,
                            `${cx.toFixed(1)},${(cy + R).toFixed(1)}`,
                            `${(cx - W / 2).toFixed(1)},${(cy + R / 2).toFixed(1)}`,
                            `${(cx - W / 2).toFixed(1)},${(cy - R / 2).toFixed(1)}`,
                          ].join(' ');

                        return (
                          <div key={activeComputer?.id || 'single-band'} className="hexbin-band-row">
                            <button
                              type="button"
                              className="hexbin-band-pill"
                              style={{
                                backgroundColor: palette.pillBg,
                                color: palette.pillText,
                              }}
                              onClick={() => {
                                if (primaryScan) {
                                  setSelectedScanId(primaryScan.id);
                                  setDetailView('history');
                                  setActiveTab('scans');
                                }
                              }}
                              title={`${activeComputer?.hostname || ''} · Predominant: ${palette.label} (+${activeComputerOpSummary.createCount} Created, ~${activeComputerOpSummary.updateCount} Updated, -${activeComputerOpSummary.deleteCount} Deleted)`}
                            >
                              <span className="hexbin-band-pill-text">
                                {activeComputer?.hostname} · {palette.label} (+{activeComputerOpSummary.createCount} / ~{activeComputerOpSummary.updateCount} / -{activeComputerOpSummary.deleteCount})
                              </span>
                            </button>

                            <div className="hexbin-svg-wrap">
                              <svg
                                className="hexbin-band-svg"
                                viewBox={`0 0 ${svgWidth} ${svgHeight}`}
                                preserveAspectRatio="xMidYMid meet"
                                role="img"
                                aria-label={`Hexbin activity for ${activeComputer?.hostname || 'selected computer'}`}
                              >
                                {cells.map((cell) => {
                                  const cx = padX + cell.col * dx + (cell.row % 2 === 1 ? dx * 0.5 : 0);
                                  const cy = padY + cell.row * dy;
                                  const cellPalette =
                                    activeHexbinPalettes[cell.op] || activeHexbinPalettes.create;
                                  const style = cellPalette.levels[cell.pct] || cellPalette.levels[6];
                                  const targetScan =
                                    hexbinVisibleScans[(cell.row + cell.col) % hexbinVisibleScans.length] || primaryScan;
                                  return (
                                    <g
                                      key={`${cell.row}-${cell.col}`}
                                      className="hexbin-cell-group"
                                      onClick={() => {
                                        if (targetScan) {
                                          setSelectedScanId(targetScan.id);
                                          setDetailView(cell.pct >= 86 ? 'history' : 'entries');
                                          setActiveTab('scans');
                                        }
                                      }}
                                    >
                                      <title>
                                        {`${activeComputer?.hostname || ''} · ${targetScan?.root_path || ''}\nHexagon Predominant Operation: ${cellPalette.label} (+${cell.added} Created / ~${cell.modified} Updated / -${cell.deleted} Deleted)\nActivity Density: ${cell.pct}%`}
                                      </title>
                                      <polygon
                                        points={hexPoints(cx, cy)}
                                        fill={style.fill}
                                        stroke={style.stroke}
                                        strokeWidth={cell.pct === 100 ? '1.5' : '1'}
                                      />
                                      <text
                                        x={cx.toFixed(1)}
                                        y={(cy + 3).toFixed(1)}
                                        textAnchor="middle"
                                        fill={style.text}
                                        fontFamily="var(--font-mono)"
                                        fontSize={cell.pct === 100 ? '8' : cell.pct === 6 ? '7.2' : '7.6'}
                                        fontWeight={cell.pct >= 86 ? '700' : '600'}
                                      >
                                        {cell.pct}%
                                      </text>
                                    </g>
                                  );
                                })}
                              </svg>
                            </div>
                          </div>
                        );
                      })()}
                    </div>
                  </div>
                )}
              </section>
            </div>

            <div className="dashboard-stack">
              {dashboardFileTypeStats.length > 0 && (
                <section className="surface-panel" aria-label="Computer File Type Composition">
                  <div className="panel-header">
                    <h2 className="panel-title">Storage Composition ({activeComputer?.hostname || 'Host'})</h2>
                    <button
                      type="button"
                      className="btn"
                      onClick={() => {
                        if (activeComputer) setFileTypeScope(`comp:${activeComputer.id}`);
                        setActiveTab('filetypes');
                      }}
                    >
                      Explore File Types →
                    </button>
                  </div>
                  <div className="filetype-bar-section">
                    <div className="stacked-dist-bar">
                      {dashboardFileTypeStats.slice(0, 10).map((st) => (
                        <div
                          key={st.ext}
                          tabIndex={0}
                          role="button"
                          aria-label={`${st.ext} (${st.pctBytes.toFixed(1)}%)`}
                          className="stacked-dist-segment"
                          style={{
                            width: `${Math.max(st.pctBytes, 1)}%`,
                            background: st.style.swatch,
                          }}
                          title={`${st.ext} (${st.category}) · ${formatBytes(st.bytes)} (${st.pctBytes.toFixed(1)}%)`}
                          onClick={() => {
                            if (activeComputer) setFileTypeScope(`comp:${activeComputer.id}`);
                            setSelectedFileTypeExt(st.ext);
                            setActiveTab('filetypes');
                          }}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault();
                              if (activeComputer) setFileTypeScope(`comp:${activeComputer.id}`);
                              setSelectedFileTypeExt(st.ext);
                              setActiveTab('filetypes');
                            }
                          }}
                        >
                          {st.pctBytes >= 8 ? st.ext : ''}
                        </div>
                      ))}
                    </div>
                    <div className="stacked-dist-legend">
                      {dashboardFileTypeStats.slice(0, 6).map((st) => (
                        <button
                          key={st.ext}
                          type="button"
                          className="legend-pill"
                          onClick={() => {
                            if (activeComputer) setFileTypeScope(`comp:${activeComputer.id}`);
                            setSelectedFileTypeExt(st.ext);
                            setActiveTab('filetypes');
                          }}
                        >
                          <span className="ext-Swatch" style={{ background: st.style.swatch }} />
                          <span>{st.ext}</span>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {formatBytes(st.bytes)} ({st.pctBytes.toFixed(0)}%)
                          </span>
                        </button>
                      ))}
                    </div>
                  </div>
                </section>
              )}

              <section className="surface-panel" aria-label="Tracked Computer & Drives Overview">
                <div className="panel-header">
                  <h2 className="panel-title">
                    Selected Computer &amp; Drives ({activeComputerDrivesCount})
                  </h2>
                  <button
                    id="dash-open-scans-btn"
                    type="button"
                    className="btn"
                    onClick={() => setActiveTab('scans')}
                  >
                    Open Explorer
                  </button>
                </div>

                {filteredDashboardComputers.length === 0 ? (
                  <div className="empty-state">
                    {computers.length === 0
                      ? 'No computers have uploaded scans yet. Run a scan from the CLI to populate this dashboard.'
                      : 'No computers match the current search query.'}
                  </div>
                ) : (
                  <div className="computer-overview-list">
                    {filteredDashboardComputers.map((comp) => {
                      const sharePct =
                        totalBytesAcrossScans > 0
                          ? Math.min(100, Math.max(3, Math.round((comp.total_bytes / totalBytesAcrossScans) * 100)))
                          : 100;
                      return (
                        <div key={comp.id} className="computer-overview-card">
                          <div className="computer-title-row">
                            <div>
                              <span className="computer-name">{comp.hostname}</span>
                              {comp.id && comp.id !== comp.hostname && (
                                <span
                                  className="mono-cell"
                                  style={{ marginLeft: '0.5rem', color: 'var(--text-muted)' }}
                                >
                                  ({comp.id.slice(0, 12)})
                                </span>
                              )}
                            </div>
                            <span className="tag-badge">
                              {(comp.directories || []).length} dir(s) · {formatBytes(comp.total_bytes)}
                            </span>
                          </div>

                          {(comp.drives || []).length > 0 && (
                            <div className="drive-chip-row">
                              {comp.drives.map((drv) => (
                                <span key={drv} className="drive-chip">
                                  Drive: {drv}
                                </span>
                              ))}
                            </div>
                          )}

                          <div className="storage-bar-track" title={`${sharePct}% of indexed storage`}>
                            <div className="storage-bar-fill" style={{ width: `${sharePct}%` }} />
                          </div>

                          <div className="scan-meta-row">
                            <span>
                              {(comp.total_files || 0).toLocaleString()} files · {(comp.total_dirs || 0).toLocaleString()} dirs
                            </span>
                            <div className="toolbar-group">
                              <button
                                type="button"
                                className="btn"
                                onClick={() => {
                                  setSelectedComputerId(comp.id);
                                  setActiveTab('tree');
                                }}
                              >
                                Directory Tree
                              </button>
                              <button
                                type="button"
                                className="btn"
                                onClick={() => {
                                  setSelectedComputerId(comp.id);
                                  setActiveTab('treemap');
                                }}
                              >
                                Treemap
                              </button>
                              <button
                                type="button"
                                className="btn"
                                onClick={() => {
                                  if ((comp.directories || []).length > 0) {
                                    setSelectedScanId(comp.directories[0].id);
                                  }
                                  setActiveTab('scans');
                                }}
                              >
                                Inspect →
                              </button>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </section>

              <section className="surface-panel" aria-label="Top Duplicate Groups">
                <div className="panel-header">
                  <h2 className="panel-title">Top Duplicate Space Offenders</h2>
                  <button
                    id="dash-open-dups-btn"
                    type="button"
                    className="btn"
                    onClick={() => setActiveTab('duplicates')}
                  >
                    View All ({duplicates.length})
                  </button>
                </div>

                {filteredDuplicates.length === 0 ? (
                  <div className="empty-state">
                    No duplicate files detected across stored directory snapshots.
                  </div>
                ) : (
                  <div className="table-wrapper">
                    <table className="data-table">
                      <thead>
                        <tr>
                          {renderSortableTh('dashDups', 'name', 'File & Hash', 'asc')}
                          {renderSortableTh('dashDups', 'count', 'Copies', 'desc')}
                          {renderSortableTh('dashDups', 'wasted_bytes', 'Wasted', 'desc')}
                        </tr>
                      </thead>
                      <tbody>
                        {sortedDashboardDuplicates.slice(0, 5).map((g) => {
                          const dupKey = `${g.hash}-${g.size}`;
                          const firstName = g.files?.[0]?.name || g.hash.slice(0, 12);
                          return (
                            <tr
                              key={dupKey}
                              tabIndex={0}
                              className="tree-row"
                              onClick={() => {
                                setSelectedDupKey(dupKey);
                                setActiveTab('duplicates');
                              }}
                              onKeyDown={(e) => {
                                if (e.key === 'Enter' || e.key === ' ') {
                                  e.preventDefault();
                                  setSelectedDupKey(dupKey);
                                  setActiveTab('duplicates');
                                }
                              }}
                            >
                              <td>
                                <span className="mono-cell" style={{ fontWeight: 600, marginRight: '0.45rem' }}>
                                  {firstName}
                                </span>
                                <span className="hash-chip">{g.hash.slice(0, 10)}</span>
                              </td>
                              <td className="mono-cell">{g.count}x</td>
                              <td className="mono-cell">{formatBytes(g.wasted_bytes)}</td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </section>
            </div>
          </div>
          </div>
        )}

        {activeTab === 'tree' && (
          <div className={`op-workspace-split ${opDrawerOpen ? '' : 'drawer-collapsed'}`} id="windir-tree-workspace">
            {opDrawerOpen && (
              <aside className="op-sidebar-panel" aria-label="Tracked Computer & Subtree Operations">
                <div className="op-sidebar-header">
                  <div>
                    <div className="op-sidebar-title">Tracked Subtrees</div>
                    <div className="op-sidebar-subtitle">
                      {activeComputer ? `${activeComputer.hostname} · ${(activeComputer.directories || []).length} snapshot(s)` : 'Select a host'}
                    </div>
                  </div>
                  <select
                    id="windir-computer-select"
                    className="op-compact-select"
                    value={activeComputer?.id || ''}
                    onChange={(e) => {
                      setSelectedComputerId(e.target.value);
                      setFocusedTreeKey('');
                      setHighlightedExt('');
                      setSelectedTreemapFile(null);
                    }}
                  >
                    {effectiveComputers.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.hostname}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="op-card-list">
                  {allDirectoryNodes.slice(0, 18).map(({ node, depth }, idx) => {
                    const isFocused = focusedTreeNode?.key === node.key;
                    const pctHost = Math.round(node.pctOfRoot || 0);
                    return renderOpCard({
                      id: `tree-op-card-${idx}`,
                      cardKey: node.key,
                      selected: isFocused,
                      badgeTone: node.kind === 'computer' ? 'red' : isFocused ? 'blue' : 'dark',
                      title: node.name,
                      versionBadge: node.kind === 'computer' ? 'HOST' : node.kind === 'root' ? 'ROOT' : `L${depth}`,
                      pct: pctHost,
                      leftTitle: activeComputer?.hostname || 'Host',
                      leftStatus: node.kind === 'computer' ? 'Root Host' : 'Active Tree',
                      leftStatusTone: 'ok',
                      rightTitle: node.driveId || activeComputer?.drives?.[0] || 'volume-0',
                      rightDelta: formatBytes(node.subtreeBytes),
                      pills: [
                        `◫ ${node.fileCount || 0}f`,
                        `▣ ${node.dirCount || 0}d`,
                        node.hash ? `#${node.hash.slice(0, 6)}` : 'merkle',
                      ],
                      shareBadge: `${pctHost}%`,
                      onClick: () => {
                        setFocusedTreeKey(node.key);
                        setSelectedTreemapFile(null);
                      },
                    });
                  })}
                </div>
              </aside>
            )}

            <div className="op-main-stage">
              {!winDirRoot || (activeComputer?.directories || []).length === 0 ? (
                <div className="surface-panel empty-state" style={{ flex: 1 }}>
                  <p>No directory snapshots available to build a computer directory tree.</p>
                  <div className="code-hint">
                    go run . scan -R --server=http://localhost:8080 ./pkg
                  </div>
                </div>
              ) : (
                <div className="windir-top-split" style={{ flex: 1 }}>
                  <section className="surface-panel" aria-label="WinDirStat Hierarchical Directory Tree">
                    <div className="panel-header">
                      <div>
                        <h2 className="panel-title">
                          Directory Tree — {activeComputer?.hostname}
                        </h2>
                        <div
                          className="mono-cell"
                          style={{ fontSize: '0.74rem', color: 'var(--text-secondary)', marginTop: '0.18rem' }}
                        >
                          Focused Subtree: <strong>{focusedTreeNode?.name}</strong> ({formatBytes(focusedTreeNode?.subtreeBytes)})
                        </div>
                      </div>
                      <div className="toolbar-group">
                        {highlightedExt && (
                          <button
                            type="button"
                            className="btn"
                            onClick={() => setHighlightedExt('')}
                          >
                            Clear {highlightedExt} ✕
                          </button>
                        )}
                        <span className="tag-badge">
                          {filteredVisibleTreeRows.length.toLocaleString()} visible row(s)
                        </span>
                      </div>
                    </div>

                    <div className="tree-table-scroll tree-page-full-scroll">
                      <table className="data-table" id="windir-tree-table">
                        <thead>
                          <tr>
                            {renderSortableTh('tree', 'name', 'Name', 'asc')}
                            {renderSortableTh('tree', 'pctOfParent', 'Subtree %', 'desc')}
                            {renderSortableTh('tree', 'pctOfRoot', '% Host', 'desc')}
                            {renderSortableTh('tree', 'subtreeBytes', 'Size', 'desc')}
                            {renderSortableTh('tree', 'fileCount', 'Files', 'desc')}
                            {renderSortableTh('tree', 'dirCount', 'Subdirs', 'desc')}
                            {renderSortableTh('tree', 'hash', 'Merkle Hash / Drive', 'asc')}
                          </tr>
                        </thead>
                        <tbody>
                          {filteredVisibleTreeRows.map(({ node, depth }) => {
                            const isExpanded = Boolean(expandedTreeNodes[node.key]);
                            const isFocused = focusedTreeNode?.key === node.key;
                            const isSelectedFile = selectedTreemapFile?.key === node.key;
                            const isExtMatch = Boolean(highlightedExt && !node.isDir && node.ext === highlightedExt);
                            const pctBar = Math.min(100, Math.max(0, node.pctOfParent || 0));

                            return (
                              <tr
                                key={node.key}
                                tabIndex={0}
                                className={`tree-row ${isFocused || isSelectedFile || isExtMatch ? 'focused' : ''}`}
                                onClick={() => {
                                  if (node.isDir) {
                                    setFocusedTreeKey(node.key);
                                    setSelectedTreemapFile(null);
                                  } else {
                                    setSelectedTreemapFile(node);
                                    if (node.ext) setHighlightedExt(node.ext);
                                  }
                                }}
                                onDoubleClick={() => {
                                  if (node.isDir) {
                                    handleToggleTreeExpand(node.key);
                                  }
                                }}
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter' || e.key === ' ') {
                                    e.preventDefault();
                                    if (node.isDir) {
                                      setFocusedTreeKey(node.key);
                                      setSelectedTreemapFile(null);
                                    } else {
                                      setSelectedTreemapFile(node);
                                      if (node.ext) setHighlightedExt(node.ext);
                                    }
                                  }
                                }}
                              >
                                <td>
                                  <div
                                    className="tree-cell-name"
                                    style={{ paddingLeft: `${depth * 18}px` }}
                                  >
                                    {node.isDir ? (
                                      <button
                                        type="button"
                                        className="tree-Caret-btn"
                                        onClick={(e) => handleToggleTreeExpand(node.key, e)}
                                        aria-label={isExpanded ? 'Collapse folder' : 'Expand folder'}
                                      >
                                        {isExpanded ? '▾' : '▸'}
                                      </button>
                                    ) : (
                                      <span style={{ width: '20px', display: 'inline-block' }} />
                                    )}

                                    <span
                                      className={`type-badge ${node.isDir ? 'dir' : 'file'}`}
                                    >
                                      {node.kind === 'computer'
                                        ? 'HOST'
                                        : node.kind === 'root'
                                          ? 'ROOT'
                                          : node.isDir
                                            ? 'DIR'
                                            : 'FILE'}
                                    </span>

                                    <span
                                      className="mono-cell"
                                      style={{
                                        fontWeight: node.isDir ? 600 : 400,
                                        color: 'var(--text-primary)',
                                      }}
                                    >
                                      {node.name}
                                    </span>
                                  </div>
                                </td>
                                <td>
                                  <div className="subtree-pct-wrap">
                                    <div className="subtree-pct-track">
                                      <div
                                        className="subtree-pct-fill"
                                        style={{ width: `${pctBar.toFixed(1)}%` }}
                                      />
                                    </div>
                                    <span className="subtree-pct-text">
                                      {pctBar.toFixed(1)}%
                                    </span>
                                  </div>
                                </td>
                                <td className="mono-cell">
                                  {(node.pctOfRoot || 0).toFixed(1)}%
                                </td>
                                <td className="mono-cell" style={{ fontWeight: node.isDir ? 600 : 400 }}>
                                  {formatBytes(node.subtreeBytes)}
                                </td>
                                <td className="mono-cell">
                                  {node.isDir ? (node.fileCount || 0).toLocaleString() : '1'}
                                </td>
                                <td className="mono-cell">
                                  {node.isDir ? (node.dirCount || 0).toLocaleString() : '—'}
                                </td>
                                <td>
                                  {node.kind === 'root' && node.driveId ? (
                                    <span className="drive-chip">{node.driveId}</span>
                                  ) : node.hash ? (
                                    <span className="hash-chip">{node.hash.slice(0, 10)}</span>
                                  ) : (
                                    <span className="mono-cell" style={{ color: 'var(--text-muted)' }}>—</span>
                                  )}
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  </section>

                  <section className="surface-panel" aria-label="Extension Statistics Breakdown">
                    <div className="panel-header">
                      <h2 className="panel-title">
                        Extension List ({extensionStats.length})
                      </h2>
                      <span className="tag-badge">
                        {focusedLeafFiles.length.toLocaleString()} files
                      </span>
                    </div>

                    <div className="tree-table-scroll tree-page-full-scroll">
                      <table className="data-table" id="windir-extensions-table">
                        <thead>
                          <tr>
                            {renderSortableTh('treeExts', 'ext', 'Extension', 'asc')}
                            {renderSortableTh('treeExts', 'color', 'Color', 'asc')}
                            {renderSortableTh('treeExts', 'bytes', 'Bytes', 'desc')}
                            {renderSortableTh('treeExts', 'pct', '%', 'desc')}
                            {renderSortableTh('treeExts', 'count', 'Files', 'desc')}
                          </tr>
                        </thead>
                        <tbody>
                          {extensionStats.map((st) => {
                            const isSelectedExt = highlightedExt === st.ext;
                            return (
                              <tr
                                key={st.ext}
                                tabIndex={0}
                                role="button"
                                aria-pressed={isSelectedExt}
                                className={`tree-row ${isSelectedExt ? 'focused' : ''}`}
                                onClick={() =>
                                  setHighlightedExt((prev) => (prev === st.ext ? '' : st.ext))
                                }
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter' || e.key === ' ') {
                                    e.preventDefault();
                                    setHighlightedExt((prev) => (prev === st.ext ? '' : st.ext));
                                  }
                                }}
                              >
                                <td className="mono-cell" style={{ fontWeight: 600 }}>
                                  {st.ext}
                                </td>
                                <td>
                                  <span
                                    className="ext-Swatch"
                                    style={{ background: st.style.swatch }}
                                  />
                                </td>
                                <td className="mono-cell">{formatBytes(st.bytes)}</td>
                                <td className="mono-cell">{st.pct.toFixed(1)}%</td>
                                <td className="mono-cell">{st.count.toLocaleString()}</td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  </section>
                </div>
              )}

              <div className="op-floating-bottom-bar" role="tablist" aria-label="Computer Selector">
                {effectiveComputers.map((c) => {
                  const isCurrent = activeComputer?.id === c.id;
                  return (
                    <button
                      key={c.id}
                      type="button"
                      role="tab"
                      aria-selected={isCurrent}
                      className={`op-floor-tab ${isCurrent ? 'active' : ''}`}
                      onClick={() => {
                        setSelectedComputerId(c.id);
                        setFocusedTreeKey('');
                        setHighlightedExt('');
                        setSelectedTreemapFile(null);
                      }}
                    >
                      {isCurrent && <span className="op-floor-dot" aria-hidden="true" />}
                      <span>{c.hostname}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'treemap' && (
          <div className={`op-workspace-split ${opDrawerOpen ? '' : 'drawer-collapsed'}`} id="treemap-workspace">
            {opDrawerOpen && (
              <aside className="op-sidebar-panel" aria-label="Reported 3D Modules & Subtrees">
                <div className="op-sidebar-header">
                  <div>
                    <div className="op-sidebar-title">Reported Modules</div>
                    <div className="op-sidebar-subtitle">
                      {focusedTreeNode ? `${focusedTreeNode.name} (${isoCushionBlocks.length} modules)` : '3D storage modules'}
                    </div>
                  </div>
                  <select
                    id="treemap-computer-select"
                    className="op-compact-select"
                    value={activeComputer?.id || ''}
                    onChange={(e) => {
                      setSelectedComputerId(e.target.value);
                      setFocusedTreeKey('');
                      setHighlightedExt('');
                      setSelectedTreemapFile(null);
                    }}
                  >
                    {effectiveComputers.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.hostname}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="op-sidebar-subfilter">
                  <select
                    id="treemap-subtree-select"
                    className="op-compact-select full"
                    value={focusedTreeNode?.key || ''}
                    onChange={(e) => {
                      setFocusedTreeKey(e.target.value);
                      setSelectedTreemapFile(null);
                    }}
                  >
                    {allDirectoryNodes.map(({ node, depth }) => (
                      <option key={node.key} value={node.key}>
                        {`${'\u00A0\u00A0'.repeat(depth)}${node.kind === 'computer' ? '[HOST] ' : node.kind === 'root' ? '[ROOT] ' : ''}${node.name} (${formatBytes(node.subtreeBytes)})`}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="op-card-list">
                  {isoCushionBlocks.map((block, idx) => {
                    const { item, index, bytes, pct, dominantExt } = block;
                    const defaultActiveIdx = Math.min(2, isoCushionBlocks.length - 1);
                    const isSelected = selectedTreemapFile
                      ? selectedTreemapFile.key === item.key
                      : idx === defaultActiveIdx;
                    return renderOpCard({
                      id: `treemap-op-card-${index}`,
                      cardKey: item.key,
                      selected: isSelected,
                      badgeTone: idx === 0 ? 'red' : isSelected ? 'blue' : 'dark',
                      title: item.name || `MODULE #${index}`,
                      versionBadge: item.isDir ? 'DIR' : dominantExt,
                      pct,
                      leftTitle: activeComputer?.hostname || 'Host',
                      leftStatus: item.isDir ? 'Subtree Ready' : 'Verified Hash',
                      leftStatusTone: 'ok',
                      rightTitle: item.driveId || activeComputer?.drives?.[0] || `NODE-0${index}`,
                      rightDelta: `+${formatBytes(bytes)}`,
                      pills: [
                        item.isDir ? `◫ ${item.fileCount || 0} files` : `◫ ${dominantExt}`,
                        item.isDir ? `▣ ${item.dirCount || 0} dirs` : `#${(item.hash || 'crc32').slice(0, 6)}`,
                      ],
                      shareBadge: `${Math.round(pct)}%`,
                      onClick: () => setSelectedTreemapFile(item),
                      onDoubleClick: () => {
                        if (item.isDir) {
                          setFocusedTreeKey(item.key);
                          setSelectedTreemapFile(null);
                        }
                      },
                    });
                  })}
                </div>
              </aside>
            )}

            <div className="op-main-stage">
              {!winDirRoot || (activeComputer?.directories || []).length === 0 ? (
                <div className="surface-panel empty-state" style={{ flex: 1 }}>
                  <p>No directory snapshots available to render a 3D Cushion Treemap.</p>
                  <div className="code-hint">
                    go run . scan -R --server=http://localhost:8080 ./pkg
                  </div>
                </div>
              ) : (
                <div className="op-full-stage iso-stage-wrapper" style={{ flex: 1 }}>
                  {isoCushionBlocks.length === 0 ? (
                    <div className="surface-panel empty-state" style={{ flex: 1 }}>
                      No non-empty items in the currently focused folder subtree.
                    </div>
                  ) : (
                    <div
                      className="treemap-container treemap-page-full-canvas iso-cushion-stage"
                      id="windir-treemap-canvas"
                      onWheel={(e) => {
                        e.preventDefault();
                        const delta = e.deltaY < 0 ? 0.1 : -0.1;
                        setIsoZoom((z) => Math.min(1.85, Math.max(0.65, +(z + delta).toFixed(2))));
                      }}
                      onMouseDown={(e) => {
                        if (e.button !== 0 || e.target.closest('button') || e.target.closest('aside')) return;
                        panDragRef.current = {
                          active: true,
                          startX: e.clientX,
                          startY: e.clientY,
                          origX: isoPan.x,
                          origY: isoPan.y,
                          moved: false,
                        };
                      }}
                      onMouseMove={(e) => {
                        if (!panDragRef.current.active) return;
                        const dx = e.clientX - panDragRef.current.startX;
                        const dy = e.clientY - panDragRef.current.startY;
                        if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
                          panDragRef.current.moved = true;
                        }
                        setIsoPan({
                          x: Math.max(-280, Math.min(280, panDragRef.current.origX + dx)),
                          y: Math.max(-220, Math.min(220, panDragRef.current.origY + dy)),
                        });
                      }}
                      onMouseUp={() => {
                        panDragRef.current.active = false;
                      }}
                      onMouseLeave={() => {
                        panDragRef.current.active = false;
                      }}
                    >
                      {(() => {
                        const CX = 510 + isoPan.x;
                        const CY = 108 + isoPan.y;
                        const SCALE = 4.35 * isoZoom;
                        const COS30 = 0.8660254;
                        const SIN30 = 0.5;
                        const isoPt = (u, v, z = 0) => ({
                          x: CX + (u - v) * SCALE * COS30,
                          y: CY + (u + v) * SCALE * SIN30 - z,
                        });
                        const polyPoints = (pts) =>
                          pts.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ');

                        const defaultActiveIdx = Math.min(2, isoCushionBlocks.length - 1);
                        const activeBlock =
                          isoCushionBlocks.find((b) => b.item.key === selectedTreemapFile?.key) ||
                          isoCushionBlocks[defaultActiveIdx] ||
                          null;
                        const activeItem = activeBlock?.item || null;
                        const searchQuery = entryFilter.trim().toLowerCase();

                        const gridTicks = [-20, 0, 20, 40, 60, 80, 100, 120];

                        return (
                          <>
                            <svg
                              className="iso-cushion-svg"
                              viewBox="0 0 1020 620"
                              preserveAspectRatio="xMidYMid meet"
                              role="img"
                              aria-label="3D Isometric Cushion Map"
                            >
                              <defs>
                                <pattern
                                  id="iso-hatch-pattern"
                                  width="6"
                                  height="6"
                                  patternTransform="rotate(35)"
                                  patternUnits="userSpaceOnUse"
                                >
                                  <line
                                    x1="0"
                                    y1="0"
                                    x2="0"
                                    y2="6"
                                    stroke={darkMode ? '#243042' : '#cfd3d1'}
                                    strokeWidth="1.4"
                                  />
                                </pattern>
                                <linearGradient id="iso-top-grad" x1="0%" y1="0%" x2="100%" y2="100%">
                                  <stop offset="0%" stopColor={darkMode ? '#1e293b' : '#f8f9f8'} />
                                  <stop offset="100%" stopColor={darkMode ? '#162030' : '#ecefee'} />
                                </linearGradient>
                                <linearGradient id="iso-top-selected-grad" x1="0%" y1="0%" x2="100%" y2="100%">
                                  <stop offset="0%" stopColor={darkMode ? '#29384f' : '#ffffff'} />
                                  <stop offset="100%" stopColor={darkMode ? '#1f2b3e' : '#f2f4f3'} />
                                </linearGradient>
                                <linearGradient id="iso-left-wall-grad" x1="0%" y1="0%" x2="0%" y2="100%">
                                  <stop offset="0%" stopColor={darkMode ? '#141c29' : '#e3e6e5'} />
                                  <stop offset="100%" stopColor={darkMode ? '#0e141e' : '#d9dddc'} />
                                </linearGradient>
                                <linearGradient id="iso-right-wall-grad" x1="0%" y1="0%" x2="0%" y2="100%">
                                  <stop offset="0%" stopColor={darkMode ? '#192333' : '#ecefed'} />
                                  <stop offset="100%" stopColor={darkMode ? '#121926' : '#e1e5e3'} />
                                </linearGradient>
                                <filter id="iso-drop-shadow" x="-15%" y="-15%" width="130%" height="140%">
                                  <feDropShadow
                                    dx="0"
                                    dy="8"
                                    stdDeviation="6"
                                    floodColor={darkMode ? '#020617' : '#1e242b'}
                                    floodOpacity={darkMode ? '0.45' : '0.1'}
                                  />
                                </filter>
                              </defs>

                              {/* Technical Isometric Floor Grid */}
                              <g className="iso-floor-grid">
                                {gridTicks.map((t) => {
                                  const a1 = isoPt(t, -25, 0);
                                  const a2 = isoPt(t, 125, 0);
                                  const b1 = isoPt(-25, t, 0);
                                  const b2 = isoPt(125, t, 0);
                                  return (
                                    <g key={`grid-${t}`}>
                                      <line
                                        x1={a1.x}
                                        y1={a1.y}
                                        x2={a2.x}
                                        y2={a2.y}
                                        stroke={darkMode ? '#1e293b' : '#dcdfdd'}
                                        strokeWidth="1"
                                      />
                                      <line
                                        x1={b1.x}
                                        y1={b1.y}
                                        x2={b2.x}
                                        y2={b2.y}
                                        stroke={darkMode ? '#1e293b' : '#dcdfdd'}
                                        strokeWidth="1"
                                      />
                                    </g>
                                  );
                                })}

                                {/* Hatched Sub-Floor Reference Pads (matching bottom of cushion-map.jpg) */}
                                <polygon
                                  points={polyPoints([
                                    isoPt(8, 102, 0),
                                    isoPt(44, 102, 0),
                                    isoPt(44, 124, 0),
                                    isoPt(8, 124, 0),
                                  ])}
                                  fill="url(#iso-hatch-pattern)"
                                  stroke={darkMode ? '#243042' : '#cfd3d1'}
                                  strokeWidth="1"
                                />
                                <polygon
                                  points={polyPoints([
                                    isoPt(102, 8, 0),
                                    isoPt(124, 8, 0),
                                    isoPt(124, 44, 0),
                                    isoPt(102, 44, 0),
                                  ])}
                                  fill="url(#iso-hatch-pattern)"
                                  stroke={darkMode ? '#243042' : '#cfd3d1'}
                                  strokeWidth="1"
                                />
                              </g>

                              {/* 3D Isometric Extruded Modules (Painter's Order) */}
                              {isoCushionBlocks.map((block) => {
                                const {
                                  item,
                                  x,
                                  y,
                                  w,
                                  h,
                                  index,
                                  bytes,
                                  pct,
                                  ratioToMax,
                                  litDots,
                                  dominantExt,
                                  style,
                                } = block;

                                const minDim = Math.min(w, h);
                                const gap = minDim > 14 ? 3.2 : minDim > 8 ? 2.0 : minDim * 0.14;
                                const u0 = x + gap;
                                const v0 = y + gap;
                                const u1 = Math.max(u0 + 2.5, x + w - gap);
                                const v1 = Math.max(v0 + 2.5, y + h - gap);
                                const du = u1 - u0;
                                const dv = v1 - v0;

                                const isSelected = activeItem?.key === item.key;
                                const isExtDimmed =
                                  highlightedExt &&
                                  highlightedExt !== dominantExt &&
                                  highlightedExt !== item.ext;
                                const isSearchDimmed =
                                  searchQuery &&
                                  !item.name?.toLowerCase().includes(searchQuery) &&
                                  !item.relPath?.toLowerCase().includes(searchQuery) &&
                                  !dominantExt.toLowerCase().includes(searchQuery) &&
                                  !item.hash?.toLowerCase().includes(searchQuery);
                                const isDimmed = Boolean(isExtDimmed || isSearchDimmed);

                                const baseZ = Math.round(32 + ratioToMax * 16);
                                const hZ = isSelected ? baseZ + 7 : baseZ;

                                const T_back = isoPt(u0, v0, hZ);
                                const T_right = isoPt(u1, v0, hZ);
                                const T_front = isoPt(u1, v1, hZ);
                                const T_left = isoPt(u0, v1, hZ);

                                const B_right = isoPt(u1, v0, 0);
                                const B_front = isoPt(u1, v1, 0);
                                const B_left = isoPt(u0, v1, 0);

                                const T_center = isoPt((u0 + u1) / 2, (v0 + v1) / 2, hZ);

                                const inset = Math.min(2.6, Math.min(du, dv) * 0.16);
                                const rivetBack = isoPt(u0 + inset, v0 + inset, hZ);
                                const rivetLeft = isoPt(u0 + inset, v1 - inset, hZ);
                                const rivetFront = isoPt(u1 - inset, v1 - inset, hZ);
                                const ledRight = isoPt(u1 - inset * 1.15, v0 + inset * 1.15, hZ);

                                const stripWidth = Math.min(2.2, Math.max(0.9, dv * 0.11));
                                const S_back = isoPt(u0 + inset * 0.7, v0 + inset * 0.7, hZ);
                                const S_right = isoPt(u1 - inset * 1.9, v0 + inset * 0.7, hZ);
                                const S_front = isoPt(u1 - inset * 1.9, v0 + inset * 0.7 + stripWidth, hZ);
                                const S_left = isoPt(u0 + inset * 0.7, v0 + inset * 0.7 + stripWidth, hZ);

                                const leftWallLen = du * SCALE;
                                const rightWallLen = dv * SCALE;
                                const radarRadius = Math.min(du, dv) * SCALE * 0.28;

                                const rawLabel = (item.name || `MODULE #${index}`).toUpperCase();
                                const maxWallChars = Math.max(6, Math.floor((leftWallLen - 34) / 6.2));
                                const wallTitle =
                                  rawLabel.length > maxWallChars
                                    ? `${rawLabel.slice(0, Math.max(4, maxWallChars - 1))}…`
                                    : rawLabel;

                                const maxTopChars = Math.max(6, Math.floor((Math.min(du, dv) * SCALE) / 5.5));
                                const topName =
                                  item.name && item.name.length > maxTopChars
                                    ? `${item.name.slice(0, Math.max(4, maxTopChars - 1))}…`
                                    : item.name;

                                return (
                                  <g
                                    key={item.key}
                                    role="button"
                                    tabIndex={0}
                                    aria-label={`${item.name} (${formatBytes(bytes)})`}
                                    className={`iso-module-group ${isSelected ? 'selected' : ''} ${isDimmed ? 'dimmed' : ''}`}
                                    onClick={() => {
                                      if (panDragRef.current.moved) return;
                                      setSelectedTreemapFile(item);
                                    }}
                                    onDoubleClick={() => {
                                      if (item.isDir) {
                                        setFocusedTreeKey(item.key);
                                        setSelectedTreemapFile(null);
                                      }
                                    }}
                                    onKeyDown={(e) => {
                                      if (e.key === 'Enter' || e.key === ' ') {
                                        e.preventDefault();
                                        setSelectedTreemapFile(item);
                                      }
                                    }}
                                  >
                                    <title>
                                      {`${item.absPath || item.relPath || item.name}\nType: ${item.isDir ? `Directory Subtree (dominant ${dominantExt})` : `File (${item.ext})`}\nSize: ${formatBytes(bytes)} (${pct.toFixed(1)}%)\n${item.isDir ? 'Double-click to drill down into folder' : `Hash: ${item.hash || '—'}`}`}
                                    </title>

                                    {/* Hatched Floor Shadow Pad beneath module */}
                                    <polygon
                                      points={polyPoints([
                                        isoPt(u0 + 1.2, v0 + 1.2, 0),
                                        isoPt(u1 + 1.6, v0 + 1.2, 0),
                                        isoPt(u1 + 1.6, v1 + 1.6, 0),
                                        isoPt(u0 + 1.2, v1 + 1.6, 0),
                                      ])}
                                      fill="url(#iso-hatch-pattern)"
                                      opacity="0.85"
                                    />

                                    {/* Left-Front Extruded 3D Wall */}
                                    <polygon
                                      points={polyPoints([T_left, T_front, B_front, B_left])}
                                      fill="url(#iso-left-wall-grad)"
                                      stroke={isSelected ? '#1f2328' : '#4a4f55'}
                                      strokeWidth={isSelected ? '1.5' : '1.2'}
                                      strokeLinejoin="round"
                                    />

                                    {/* Isometric Typography & Cooling Vents on Left-Front Wall */}
                                    {leftWallLen > 48 && (
                                      <g
                                        transform={`translate(${T_left.x.toFixed(1)}, ${T_left.y.toFixed(1)}) matrix(0.8660254, 0.5, 0, 1, 0, 0)`}
                                        style={{ pointerEvents: 'none' }}
                                      >
                                        <text
                                          x="10"
                                          y={(hZ * 0.63).toFixed(1)}
                                          fill="#7a8086"
                                          fontFamily="var(--font-mono)"
                                          fontSize={leftWallLen > 95 ? '10.5' : '8.8'}
                                          fontWeight="700"
                                          letterSpacing="0.07em"
                                        >
                                          {leftWallLen > 110 ? `${wallTitle} #${index}` : `#${index} ${wallTitle}`}
                                        </text>

                                        {/* Angled Cooling Vent Hash Marks (//////) */}
                                        {leftWallLen > 82 && (
                                          <g transform={`translate(${(leftWallLen - 30).toFixed(1)}, ${(hZ * 0.52).toFixed(1)})`}>
                                            {[0, 4, 8, 12, 16, 20].map((vx) => (
                                              <line
                                                key={vx}
                                                x1={vx}
                                                y1="6"
                                                x2={vx + 2.5}
                                                y2="-3"
                                                stroke="#bfc4c2"
                                                strokeWidth="1.5"
                                                strokeLinecap="round"
                                              />
                                            ))}
                                          </g>
                                        )}
                                      </g>
                                    )}

                                    {/* Right-Front Extruded 3D Wall */}
                                    <polygon
                                      points={polyPoints([T_front, T_right, B_right, B_front])}
                                      fill="url(#iso-right-wall-grad)"
                                      stroke={isSelected ? '#1f2328' : '#4a4f55'}
                                      strokeWidth={isSelected ? '1.5' : '1.2'}
                                      strokeLinejoin="round"
                                    />

                                    {/* 5 Capacity Share LED Dots on Right-Front Wall */}
                                    {rightWallLen > 42 && (
                                      <g
                                        transform={`translate(${T_front.x.toFixed(1)}, ${T_front.y.toFixed(1)}) matrix(0.8660254, -0.5, 0, 1, 0, 0)`}
                                        style={{ pointerEvents: 'none' }}
                                      >
                                        {[0, 1, 2, 3, 4].map((dotIdx) => {
                                          const spacing = Math.min(8.5, (rightWallLen - 16) / 5);
                                          const startX = Math.max(10, rightWallLen * 0.36);
                                          const cxDot = startX + dotIdx * spacing;
                                          if (cxDot > rightWallLen - 6) return null;
                                          const isLit = dotIdx < litDots;
                                          return (
                                            <circle
                                              key={dotIdx}
                                              cx={cxDot.toFixed(1)}
                                              cy={(hZ * 0.54).toFixed(1)}
                                              r={rightWallLen > 75 ? '2.5' : '2.0'}
                                              fill={isLit ? '#d9532f' : '#cfd4d2'}
                                            />
                                          );
                                        })}
                                      </g>
                                    )}

                                    {/* Top 3D Cushion Surface */}
                                    <polygon
                                      points={polyPoints([T_back, T_right, T_front, T_left])}
                                      fill={isSelected ? 'url(#iso-top-selected-grad)' : 'url(#iso-top-grad)'}
                                      stroke={isSelected ? '#1f2328' : '#4a4f55'}
                                      strokeWidth={isSelected ? '1.65' : '1.25'}
                                      strokeLinejoin="round"
                                      filter="url(#iso-drop-shadow)"
                                    />

                                    {/* Dominant File-Type Extension Hue Bar + Corner Rivets + Status LED */}
                                    {minDim > 8 && (
                                      <g style={{ pointerEvents: 'none' }}>
                                        {du > 7 && (
                                          <polygon
                                            points={polyPoints([S_back, S_right, S_front, S_left])}
                                            fill={style.swatch}
                                            opacity={isSelected ? '0.88' : '0.68'}
                                          />
                                        )}
                                        <circle cx={rivetLeft.x} cy={rivetLeft.y} r="1.9" fill="#1e2226" />
                                        <circle cx={rivetFront.x} cy={rivetFront.y} r="1.9" fill="#1e2226" />
                                        <ellipse
                                          cx={ledRight.x}
                                          cy={ledRight.y}
                                          rx="3.7"
                                          ry="2.3"
                                          fill={style.swatch}
                                          stroke="#3a3f44"
                                          strokeWidth="0.7"
                                        />
                                      </g>
                                    )}

                                    {/* Active / Selected Radar Target Core (from SANDBOX #3 in cushion-map.jpg) */}
                                    {isSelected && radarRadius > 10 && (
                                      <g style={{ pointerEvents: 'none' }}>
                                        <ellipse
                                          cx={T_center.x}
                                          cy={T_center.y}
                                          rx={Math.min(46, radarRadius * 1.15)}
                                          ry={Math.min(26, radarRadius * 0.66)}
                                          fill="none"
                                          stroke="#d9532f"
                                          strokeWidth="1.1"
                                          strokeDasharray="2 3.5"
                                          strokeOpacity="0.45"
                                        />
                                        <ellipse
                                          cx={T_center.x}
                                          cy={T_center.y}
                                          rx={Math.min(25, radarRadius * 0.62)}
                                          ry={Math.min(14.5, radarRadius * 0.36)}
                                          fill="none"
                                          stroke="#d9532f"
                                          strokeWidth="1"
                                          strokeDasharray="1.8 3"
                                          strokeOpacity="0.32"
                                        />
                                        <ellipse
                                          cx={T_center.x}
                                          cy={T_center.y}
                                          rx="7.5"
                                          ry="4.5"
                                          fill="#d9532f"
                                        />
                                      </g>
                                    )}

                                    {/* Top-Surface Label & Byte Volume Readout */}
                                    {minDim > 11 && (
                                      <g style={{ pointerEvents: 'none' }}>
                                        <text
                                          x={T_center.x}
                                          y={isSelected && radarRadius > 10 ? T_center.y - Math.min(28, radarRadius * 0.72) : T_center.y - 3}
                                          textAnchor="middle"
                                          fill={darkMode ? '#f1f5f9' : '#22272e'}
                                          fontFamily="var(--font-mono)"
                                          fontSize={minDim > 22 ? '10.5' : '9'}
                                          fontWeight="700"
                                        >
                                          {item.isDir ? `/${topName}` : topName}
                                        </text>
                                        <text
                                          x={T_center.x}
                                          y={isSelected && radarRadius > 10 ? T_center.y + Math.min(34, radarRadius * 0.85) : T_center.y + 11}
                                          textAnchor="middle"
                                          fill={darkMode ? '#94a3b8' : '#5c646d'}
                                          fontFamily="var(--font-mono)"
                                          fontSize="8.5"
                                          fontWeight="600"
                                        >
                                          {formatBytes(bytes)} ({pct.toFixed(0)}%)
                                        </text>
                                      </g>
                                    )}
                                  </g>
                                );
                              })}
                            </svg>

                            {/* Floating Circular Dark Action Button (matching right arrow button in cushion-map.jpg) */}
                            {activeItem && (
                              <button
                                id="iso-floating-action-btn"
                                type="button"
                                className="iso-floating-arrow-btn"
                                title={
                                  activeItem.isDir
                                    ? `Drill down into ${activeItem.name} (${formatBytes(activeItem.subtreeBytes)})`
                                    : `Inspect ${activeItem.name} in Snapshot Explorer`
                                }
                                onClick={() => {
                                  if (activeItem.isDir) {
                                    setFocusedTreeKey(activeItem.key);
                                    setSelectedTreemapFile(null);
                                  } else if (activeItem.scanId) {
                                    setSelectedScanId(activeItem.scanId);
                                    setEntryFilter(activeItem.name || '');
                                    setActiveTab('scans');
                                  }
                                }}
                              >
                                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.1" strokeLinecap="round" strokeLinejoin="round">
                                  <line x1="5" y1="12" x2="19" y2="12" />
                                  <polyline points="13 6 19 12 13 18" />
                                </svg>
                              </button>
                            )}

                            {/* Floating Glass Extension Legend Overlay when toggled in top bar */}
                            {showTreemapLegend && (
                              <aside className="iso-floating-legend-card" aria-label="Treemap Extension Legend">
                                <div className="iso-floating-legend-header">
                                  <span className="iso-floating-legend-title">
                                    Extension Legend ({extensionStats.length})
                                  </span>
                                  <button
                                    type="button"
                                    className="iso-legend-close-btn"
                                    onClick={() => setShowTreemapLegend(false)}
                                    title="Close legend"
                                  >
                                    ✕
                                  </button>
                                </div>
                                <div className="iso-floating-legend-scroll">
                                  {extensionStats.map((st) => {
                                    const isSelectedExt = highlightedExt === st.ext;
                                    return (
                                      <button
                                        key={st.ext}
                                        type="button"
                                        className={`iso-legend-row ${isSelectedExt ? 'active' : ''}`}
                                        onClick={() =>
                                          setHighlightedExt((prev) => (prev === st.ext ? '' : st.ext))
                                        }
                                      >
                                        <span className="iso-legend-ext">
                                          <span
                                            className="ext-Swatch"
                                            style={{ background: st.style.swatch }}
                                          />
                                          <span>{st.ext}</span>
                                        </span>
                                        <span className="iso-legend-meta">
                                          {formatBytes(st.bytes)} ({st.pct.toFixed(0)}%)
                                        </span>
                                      </button>
                                    );
                                  })}
                                </div>
                              </aside>
                            )}

                            {/* Bottom-Left Floating Selection / Breadcrumb HUD Pill */}
                            <div className="iso-floating-hud-left" id="treemap-selection-hud">
                              {activeBlock?.style?.swatch && (
                                <span
                                  className="ext-Swatch"
                                  style={{ background: activeBlock.style.swatch }}
                                  aria-hidden="true"
                                />
                              )}
                              <span className="iso-hud-badge">
                                {activeItem?.isDir ? 'DIR' : activeItem?.ext || 'NODE'}
                              </span>
                              <span className="iso-hud-path" title={activeItem?.absPath || activeItem?.relPath || activeItem?.name}>
                                {activeItem
                                  ? `${activeItem.name} · ${formatBytes(activeItem.subtreeBytes || activeItem.size)}${
                                      activeItem.isDir
                                        ? ` · ${activeItem.fileCount || 0}f`
                                        : activeItem.hash
                                          ? ` · #${activeItem.hash.slice(0, 8)}`
                                          : ''
                                    }`
                                  : `${focusedTreeNode?.name || 'Root'} · ${formatBytes(focusedTreeNode?.subtreeBytes)}`}
                              </span>
                              {activeItem && (
                                <button
                                  id="iso-hud-copy-btn"
                                  type="button"
                                  className="iso-hud-up-btn"
                                  onClick={() =>
                                    handleCopyText(
                                      activeItem.absPath || activeItem.relPath || activeItem.name,
                                      'iso-hud-copy'
                                    )
                                  }
                                  title="Copy selected item path"
                                >
                                  {copiedKey === 'iso-hud-copy' ? 'Copied ✓' : 'Copy'}
                                </button>
                              )}
                              {focusedTreeNode?.parentKey && (
                                <button
                                  id="iso-hud-up-parent-btn"
                                  type="button"
                                  className="iso-hud-up-btn"
                                  onClick={() => {
                                    setFocusedTreeKey(focusedTreeNode.parentKey);
                                    setSelectedTreemapFile(null);
                                  }}
                                  title="Step up one directory level"
                                >
                                  ↑ Up
                                </button>
                              )}
                              {focusedTreeNode && winDirRoot && focusedTreeNode.key !== winDirRoot.key && (
                                <button
                                  type="button"
                                  className="iso-hud-up-btn"
                                  onClick={() => {
                                    setFocusedTreeKey(winDirRoot.key);
                                    setSelectedTreemapFile(null);
                                  }}
                                  title="Return to root directory"
                                >
                                  ↑ Root
                                </button>
                              )}
                            </div>

                            {/* Bottom-Right Floating +/-/Reset Zoom Controls */}
                            <div className="iso-floating-zoom-dock" aria-label="3D Map Zoom Controls">
                              <button
                                id="iso-zoom-in-btn"
                                type="button"
                                className="iso-zoom-btn"
                                onClick={() => setIsoZoom((z) => Math.min(1.85, +(z + 0.15).toFixed(2)))}
                                title="Zoom In (+)"
                              >
                                +
                              </button>
                              <button
                                id="iso-zoom-reset-btn"
                                type="button"
                                className="iso-zoom-btn reset"
                                onClick={() => {
                                  setIsoZoom(1);
                                  setIsoPan({ x: 0, y: 0 });
                                }}
                                title="Reset Zoom & Camera Pan (100%)"
                              >
                                {Math.round(isoZoom * 100)}%
                              </button>
                              <button
                                id="iso-zoom-out-btn"
                                type="button"
                                className="iso-zoom-btn"
                                onClick={() => setIsoZoom((z) => Math.max(0.65, +(z - 0.15).toFixed(2)))}
                                title="Zoom Out (-)"
                              >
                                −
                              </button>
                            </div>
                          </>
                        );
                      })()}
                    </div>
                  )}
                </div>
              )}

              {/* Bottom-Center Floating Segmented Host Bar (matching option-bar.jpg) */}
              <div className="op-floating-bottom-bar" role="tablist" aria-label="Host Floor Selector">
                {effectiveComputers.map((c) => {
                  const isCurrent = activeComputer?.id === c.id;
                  return (
                    <button
                      key={c.id}
                      type="button"
                      role="tab"
                      aria-selected={isCurrent}
                      className={`op-floor-tab ${isCurrent ? 'active' : ''}`}
                      onClick={() => {
                        setSelectedComputerId(c.id);
                        setFocusedTreeKey('');
                        setHighlightedExt('');
                        setSelectedTreemapFile(null);
                      }}
                    >
                      {isCurrent && <span className="op-floor-dot" aria-hidden="true" />}
                      <span>{c.hostname}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'filetypes' && (
          <div className={`op-workspace-split ${opDrawerOpen ? '' : 'drawer-collapsed'}`} id="filetypes-workspace">
            {opDrawerOpen && (
              <aside className="op-sidebar-panel" aria-label="File Type Scope & Extension Cards">
                <div className="op-sidebar-header">
                  <div>
                    <div className="op-sidebar-title">Reported File Types</div>
                    <div className="op-sidebar-subtitle">
                      {fileTypeAnalysis.stats.length} extensions · {formatBytes(fileTypeAnalysis.totalBytes)}
                    </div>
                  </div>
                  <select
                    id="filetypes-scope-select"
                    className="op-compact-select"
                    value={fileTypeScope}
                    onChange={(e) => {
                      setFileTypeScope(e.target.value);
                      setSelectedFileTypeExt('');
                    }}
                  >
                    <option value="__all__">All Hosts</option>
                    <optgroup label="By Computer">
                      {effectiveComputers.map((c) => (
                        <option key={`comp:${c.id}`} value={`comp:${c.id}`}>
                          {c.hostname}
                        </option>
                      ))}
                    </optgroup>
                    <optgroup label="By Snapshot">
                      {scans.map((s) => (
                        <option key={`scan:${s.id}`} value={`scan:${s.id}`}>
                          {s.hostname}:{s.root_path}
                        </option>
                      ))}
                    </optgroup>
                  </select>
                </div>

                <div className="op-card-list">
                  {fileTypeAnalysis.stats.slice(0, 16).map((st, idx) => {
                    const pct = fileTypeMetric === 'count' ? st.pctCount : st.pctBytes;
                    const isSelected = selectedFileTypeExt === st.ext;
                    return renderOpCard({
                      id: `filetype-op-card-${idx}`,
                      cardKey: st.ext,
                      selected: isSelected,
                      badgeTone: idx === 0 ? 'red' : isSelected ? 'blue' : 'dark',
                      title: st.ext,
                      versionBadge: st.category,
                      pct,
                      leftTitle: `${st.count.toLocaleString()} file(s)`,
                      leftStatus: `Avg ${formatBytes(st.avgBytes)}`,
                      leftStatusTone: 'ok',
                      rightTitle: st.largestFile ? st.largestFile.name.slice(0, 14) : 'Max File',
                      rightDelta: formatBytes(st.bytes),
                      pills: [
                        `◫ ${st.pctCount.toFixed(0)}% count`,
                        `⛁ ${st.pctBytes.toFixed(0)}% vol`,
                      ],
                      shareBadge: `${Math.round(pct)}%`,
                      onClick: () =>
                        setSelectedFileTypeExt((prev) => (prev === st.ext ? '' : st.ext)),
                    });
                  })}
                </div>
              </aside>
            )}

            <div className="op-main-stage">
              {fileTypeAnalysis.stats.length === 0 ? (
                <div className="surface-panel empty-state" style={{ flex: 1 }}>
                  <p>No files available in the selected scope to analyze file type distribution.</p>
                </div>
              ) : (
                <div className="dashboard-stack" style={{ flex: 1 }}>
                  <section
                    className="surface-panel"
                    id="filetypes-stacked-bar-panel"
                    aria-label="File Type Stacked Distribution Bar"
                  >
                    <div className="panel-header">
                      <div>
                        <h2 className="panel-title">
                          File Type Distribution Bar ({fileTypeMetric === 'bytes' ? 'Share of Storage Bytes' : 'Share of File Count'})
                        </h2>
                        <div
                          className="mono-cell"
                          style={{ fontSize: '0.74rem', color: 'var(--text-secondary)', marginTop: '0.18rem' }}
                        >
                          {fileTypeAnalysis.stats.length} unique file extensions across{' '}
                          {fileTypeAnalysis.totalCount.toLocaleString()} files ({formatBytes(fileTypeAnalysis.totalBytes)})
                        </div>
                      </div>
                      <span className="tag-badge">
                        {selectedFileTypeExt ? `Filtered: ${selectedFileTypeExt}` : 'Click any bar segment or card to filter'}
                      </span>
                    </div>

                    <div className="filetype-bar-section">
                      <div className="stacked-dist-bar" id="filetypes-stacked-bar">
                        {fileTypeAnalysis.stats.map((st) => {
                          const pct = fileTypeMetric === 'count' ? st.pctCount : st.pctBytes;
                          const isSelected = selectedFileTypeExt === st.ext;
                          const isDimmed = selectedFileTypeExt && selectedFileTypeExt !== st.ext;
                          return (
                            <div
                              key={st.ext}
                              tabIndex={0}
                              role="button"
                              aria-pressed={isSelected}
                              aria-label={`${st.ext} (${pct.toFixed(1)}%)`}
                              className={`stacked-dist-segment ${isSelected ? 'selected' : ''} ${isDimmed ? 'dimmed' : ''}`}
                              style={{
                                width: `${Math.max(pct, 0.6)}%`,
                                background: st.style.swatch,
                              }}
                              title={`${st.ext} (${st.category})\nVolume: ${formatBytes(st.bytes)} (${st.pctBytes.toFixed(1)}%)\nFiles: ${st.count.toLocaleString()} (${st.pctCount.toFixed(1)}%)`}
                              onClick={() =>
                                setSelectedFileTypeExt((prev) => (prev === st.ext ? '' : st.ext))
                              }
                              onKeyDown={(e) => {
                                if (e.key === 'Enter' || e.key === ' ') {
                                  e.preventDefault();
                                  setSelectedFileTypeExt((prev) => (prev === st.ext ? '' : st.ext));
                                }
                              }}
                            >
                              {pct >= 5.5 ? `${st.ext} (${pct.toFixed(0)}%)` : ''}
                            </div>
                          );
                        })}
                      </div>

                      <div className="stacked-dist-legend">
                        {fileTypeAnalysis.stats.map((st) => {
                          const pct = fileTypeMetric === 'count' ? st.pctCount : st.pctBytes;
                          const isActive = selectedFileTypeExt === st.ext;
                          return (
                            <button
                              key={st.ext}
                              type="button"
                              className={`legend-pill ${isActive ? 'active' : ''}`}
                              onClick={() =>
                                setSelectedFileTypeExt((prev) => (prev === st.ext ? '' : st.ext))
                              }
                            >
                              <span
                                className="ext-Swatch"
                                style={{ background: st.style.swatch }}
                              />
                              <span>{st.ext}</span>
                              <span style={{ color: 'var(--text-secondary)' }}>
                                {fileTypeMetric === 'count'
                                  ? `${st.count} (${pct.toFixed(1)}%)`
                                  : `${formatBytes(st.bytes)} (${pct.toFixed(1)}%)`}
                              </span>
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  </section>

                  <div className="windir-top-split" style={{ flex: 1 }}>
                    <section className="surface-panel" aria-label="File Type Breakdown Table">
                      <div className="panel-header">
                        <h2 className="panel-title">
                          Breakdown by File Type ({fileTypeAnalysis.stats.length})
                        </h2>
                        <span className="tag-badge">
                          Sorted by {fileTypeMetric === 'count' ? 'File Count' : 'Total Volume'}
                        </span>
                      </div>

                      <div className="tree-table-scroll tree-page-full-scroll">
                        <table className="data-table" id="filetypes-breakdown-table">
                          <thead>
                            <tr>
                              {renderSortableTh('ftStats', 'ext', 'Extension', 'asc')}
                              {renderSortableTh('ftStats', 'category', 'Category', 'asc')}
                              {renderSortableTh(
                                'ftStats',
                                'pct',
                                `Distribution (${fileTypeMetric === 'count' ? '% Files' : '% Bytes'})`,
                                'desc'
                              )}
                              {renderSortableTh('ftStats', 'bytes', 'Total Volume', 'desc')}
                              {renderSortableTh('ftStats', 'count', 'Files', 'desc')}
                              {renderSortableTh('ftStats', 'avgBytes', 'Avg Size', 'desc')}
                              {renderSortableTh('ftStats', 'largestFile', 'Largest File', 'desc')}
                            </tr>
                          </thead>
                          <tbody>
                            {fileTypeAnalysis.stats.map((st) => {
                              const pct = fileTypeMetric === 'count' ? st.pctCount : st.pctBytes;
                              const isSelected = selectedFileTypeExt === st.ext;
                              return (
                                <tr
                                  key={st.ext}
                                  tabIndex={0}
                                  role="button"
                                  aria-pressed={isSelected}
                                  className={`tree-row ${isSelected ? 'focused' : ''}`}
                                  onClick={() =>
                                    setSelectedFileTypeExt((prev) => (prev === st.ext ? '' : st.ext))
                                  }
                                  onKeyDown={(e) => {
                                    if (e.key === 'Enter' || e.key === ' ') {
                                      e.preventDefault();
                                      setSelectedFileTypeExt((prev) => (prev === st.ext ? '' : st.ext));
                                    }
                                  }}
                                >
                                  <td className="mono-cell" style={{ fontWeight: 600 }}>
                                    <div className="tree-cell-name">
                                      <span
                                        className="ext-Swatch"
                                        style={{ background: st.style.swatch }}
                                      />
                                      <span>{st.ext}</span>
                                    </div>
                                  </td>
                                  <td>
                                    <span className="tag-badge">{st.category}</span>
                                  </td>
                                  <td>
                                    <div className="subtree-pct-wrap">
                                      <div className="dist-inline-bar-track">
                                        <div
                                          className="dist-inline-bar-fill"
                                          style={{
                                            width: `${Math.min(100, Math.max(1, pct)).toFixed(1)}%`,
                                            background: st.style.swatch,
                                          }}
                                        />
                                      </div>
                                      <span className="subtree-pct-text">
                                        {pct.toFixed(1)}%
                                      </span>
                                    </div>
                                  </td>
                                  <td className="mono-cell" style={{ fontWeight: 600 }}>
                                    {formatBytes(st.bytes)}
                                  </td>
                                  <td className="mono-cell">{st.count.toLocaleString()}</td>
                                  <td className="mono-cell">{formatBytes(st.avgBytes)}</td>
                                  <td className="mono-cell">
                                    {st.largestFile
                                      ? `${st.largestFile.name} (${formatBytes(st.largestFile.size)})`
                                      : '—'}
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                    </section>

                    <section className="surface-panel" aria-label="Files Matching Selected Extension">
                      <div className="panel-header">
                        <h2 className="panel-title">
                          {selectedFileTypeExt
                            ? `Files with Extension ${selectedFileTypeExt} (${fileTypeAnalysis.matchingFiles.length})`
                            : `Largest Files Across Scope (Top ${fileTypeAnalysis.matchingFiles.length})`}
                        </h2>
                      </div>

                      <div className="tree-table-scroll tree-page-full-scroll">
                        <table className="data-table" id="filetypes-files-table">
                          <thead>
                            <tr>
                              {renderSortableTh('ftFiles', 'path', 'Computer & Path', 'asc')}
                              {renderSortableTh('ftFiles', 'ext', 'Ext', 'asc')}
                              {renderSortableTh('ftFiles', 'size', 'Size', 'desc')}
                              {renderSortableTh('ftFiles', 'hostname', 'Action', 'asc')}
                            </tr>
                          </thead>
                          <tbody>
                            {fileTypeAnalysis.matchingFiles.map((f) => (
                              <tr key={f.key}>
                                <td className="mono-cell">
                                  <strong>{f.hostname}</strong>:{f.absPath}
                                </td>
                                <td>
                                  <span className="tag-badge">{f.ext}</span>
                                </td>
                                <td className="mono-cell">{formatBytes(f.size)}</td>
                                <td>
                                  <button
                                    type="button"
                                    className="btn"
                                    onClick={() => {
                                      setSelectedScanId(f.scanId);
                                      setEntryFilter(f.name);
                                      setActiveTab('scans');
                                    }}
                                  >
                                    Open →
                                  </button>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </section>
                  </div>
                </div>
              )}

              <div className="op-floating-bottom-bar" role="tablist" aria-label="File Type Scope Selector">
                <button
                  type="button"
                  role="tab"
                  aria-selected={fileTypeScope === '__all__'}
                  className={`op-floor-tab ${fileTypeScope === '__all__' ? 'active' : ''}`}
                  onClick={() => {
                    setFileTypeScope('__all__');
                    setSelectedFileTypeExt('');
                  }}
                >
                  {fileTypeScope === '__all__' && <span className="op-floor-dot" aria-hidden="true" />}
                  <span>All Hosts</span>
                </button>
                {effectiveComputers.map((c) => {
                  const scopeVal = `comp:${c.id}`;
                  const isCurrent = fileTypeScope === scopeVal;
                  return (
                    <button
                      key={c.id}
                      type="button"
                      role="tab"
                      aria-selected={isCurrent}
                      className={`op-floor-tab ${isCurrent ? 'active' : ''}`}
                      onClick={() => {
                        setFileTypeScope(scopeVal);
                        setSelectedFileTypeExt('');
                      }}
                    >
                      {isCurrent && <span className="op-floor-dot" aria-hidden="true" />}
                      <span>{c.hostname}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'scans' && (
          <div className={`op-workspace-split ${opDrawerOpen ? '' : 'drawer-collapsed'}`} id="scans-workspace">
            {opDrawerOpen && (
              <aside className="op-sidebar-panel" aria-label="Computers and Tracked Directories">
                <div className="op-sidebar-header">
                  <div>
                    <div className="op-sidebar-title">Reported Snapshots</div>
                    <div className="op-sidebar-subtitle">
                      {computers.length} host(s) · {filteredScansList.length} tracked dir(s)
                    </div>
                  </div>
                  <select
                    id="scans-computer-filter"
                    className="op-compact-select"
                    value={scanComputerFilter}
                    onChange={(e) => setScanComputerFilter(e.target.value)}
                  >
                    <option value="">All Hosts</option>
                    {effectiveComputers.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.hostname}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="op-card-list" id="stored-scans-list">
                  {filteredScansList.map((s) => {
                    const pctVol =
                      totalBytesAcrossScans > 0
                        ? Math.round(((s.total_bytes || 0) / totalBytesAcrossScans) * 100)
                        : 0;
                    const isSelected = selectedScanId === s.id;
                    return renderOpCard({
                      id: `scan-item-${s.id}`,
                      cardKey: s.id,
                      selected: isSelected,
                      badgeTone: (s.version || 1) > 1 ? 'red' : isSelected ? 'blue' : 'dark',
                      title: s.root_path,
                      versionBadge: `v${s.version || 1}`,
                      pct: pctVol,
                      leftTitle: s.hostname || 'Local Host',
                      leftStatus: s.tag ? `[${s.tag}]` : 'Active',
                      leftStatusTone: 'ok',
                      rightTitle: s.drive_id || 'default-drive',
                      rightDelta: formatBytes(s.total_bytes),
                      pills: [
                        `◫ ${s.total_files || 0}f`,
                        `▣ ${s.total_dirs || 0}d`,
                        `#${(s.root_hash || '').slice(0, 6)}`,
                      ],
                      shareBadge: `${pctVol}%`,
                      onClick: () => setSelectedScanId(s.id),
                    });
                  })}
                </div>
              </aside>
            )}

            <div className="op-main-stage">
              <section className="surface-panel" style={{ flex: 1 }} aria-label="Directory Snapshot and History">
                <div className="panel-header">
                  <div>
                    <h2 className="panel-title">
                      {selectedScan
                        ? `${selectedScan.hostname}:${selectedScan.root_path} (v${selectedScan.version || 1})`
                        : 'Select a directory'}
                    </h2>
                    {selectedScan && (
                      <div className="mono-cell" style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '0.2rem' }}>
                        Computer: {selectedScan.computer_id || selectedScan.hostname} · Drive: {selectedScan.drive_id || 'default-drive'} · Scan ID: {selectedScan.id}
                      </div>
                    )}
                  </div>
                  {selectedScan && (
                    <div className="toolbar-group">
                      {detailView === 'entries' && (
                        <div className="op-top-option-bar" role="toolbar" aria-label="Entry Type Filter">
                          <button
                            type="button"
                            className={`op-floor-tab ${scanEntryKindFilter === 'all' ? 'active' : ''}`}
                            onClick={() => setScanEntryKindFilter('all')}
                          >
                            <span>All</span>
                          </button>
                          <button
                            type="button"
                            className={`op-floor-tab ${scanEntryKindFilter === 'files' ? 'active' : ''}`}
                            onClick={() => setScanEntryKindFilter('files')}
                          >
                            <span>Files</span>
                          </button>
                          <button
                            type="button"
                            className={`op-floor-tab ${scanEntryKindFilter === 'dirs' ? 'active' : ''}`}
                            onClick={() => setScanEntryKindFilter('dirs')}
                          >
                            <span>Dirs</span>
                          </button>
                        </div>
                      )}
                      <button
                        type="button"
                        className="btn"
                        onClick={() => {
                          const comp = effectiveComputers.find(
                            (c) =>
                              c.id === selectedScan.computer_id ||
                              c.hostname === selectedScan.hostname ||
                              (c.directories || []).some((d) => d.id === selectedScan.id)
                          );
                          if (comp) setSelectedComputerId(comp.id);
                          setFocusedTreeKey(`scan:${selectedScan.id}`);
                          setActiveTab('tree');
                        }}
                      >
                        Directory Tree →
                      </button>
                      <button
                        type="button"
                        className="btn"
                        onClick={() => {
                          const comp = effectiveComputers.find(
                            (c) =>
                              c.id === selectedScan.computer_id ||
                              c.hostname === selectedScan.hostname ||
                              (c.directories || []).some((d) => d.id === selectedScan.id)
                          );
                          if (comp) setSelectedComputerId(comp.id);
                          setFocusedTreeKey(`scan:${selectedScan.id}`);
                          setActiveTab('treemap');
                        }}
                      >
                        3D Map →
                      </button>
                      <button
                        type="button"
                        className="btn"
                        onClick={() => {
                          setBaseScanId(selectedScan.id);
                          setActiveTab('diff');
                        }}
                      >
                        Compare →
                      </button>
                      <button
                        type="button"
                        className="hash-chip hash-copy-btn"
                        onClick={() => handleCopyText(selectedScan.root_hash, `root-${selectedScan.id}`)}
                        title="Click to copy full Merkle root hash"
                      >
                        {copiedKey === `root-${selectedScan.id}` ? 'Copied ✓' : selectedScan.root_hash}
                      </button>
                    </div>
                  )}
                </div>

                {selectedScan ? (
                  detailView === 'history' ? (
                    <div className="history-list" id="scan-history-list">
                      {(selectedScan.history || []).length === 0 ? (
                        <div className="empty-state">No change history recorded yet.</div>
                      ) : (
                        (selectedScan.history || []).map((h) => (
                          <div key={h.version} className="history-card">
                            <div className="history-card-header">
                              <div className="toolbar-group">
                                <span className="scan-id">Revision v{h.version}</span>
                                {h.tag && <span className="tag-badge">{h.tag}</span>}
                                <span className="hash-chip">
                                  {h.previous_hash
                                    ? `${h.previous_hash} → ${h.root_hash}`
                                    : h.root_hash}
                                </span>
                              </div>
                              <div className="toolbar-group">
                                <span className="diff-badge added">+{h.added} Added</span>
                                <span className="diff-badge modified">~{h.modified} Modified</span>
                                <span className="diff-badge deleted">-{h.deleted} Deleted</span>
                                <span className="mono-cell" style={{ color: 'var(--text-muted)' }}>
                                  {h.created_at ? new Date(h.created_at).toLocaleString() : ''}
                                </span>
                              </div>
                            </div>
                            {(h.changes || []).length > 0 && (
                              <div className="table-wrapper">
                                <table className="data-table">
                                  <thead>
                                    <tr>
                                      {renderSortableTh('scanHistory', 'change', 'Change', 'asc')}
                                      {renderSortableTh('scanHistory', 'is_dir', 'Type', 'asc')}
                                      {renderSortableTh('scanHistory', 'rel_path', 'Relative Path', 'asc')}
                                      {renderSortableTh('scanHistory', 'hash', 'Hash', 'asc')}
                                      {renderSortableTh('scanHistory', 'size', 'Size', 'desc')}
                                    </tr>
                                  </thead>
                                  <tbody>
                                    {[...(h.changes || [])]
                                      .sort((a, b) => {
                                        const sortCfg = tableSorts.scanHistory || { field: 'rel_path', dir: 'asc' };
                                        switch (sortCfg.field) {
                                          case 'change':
                                            return compareValues(a.change || '', b.change || '', sortCfg.dir);
                                          case 'is_dir':
                                            return compareValues(a.is_dir ? 'DIR' : 'FILE', b.is_dir ? 'DIR' : 'FILE', sortCfg.dir);
                                          case 'hash':
                                            return compareValues(
                                              a.target_hash || a.base_hash || '',
                                              b.target_hash || b.base_hash || '',
                                              sortCfg.dir
                                            );
                                          case 'size':
                                            return compareValues(
                                              a.target_size ?? a.base_size ?? 0,
                                              b.target_size ?? b.base_size ?? 0,
                                              sortCfg.dir
                                            );
                                          case 'rel_path':
                                          default:
                                            return compareValues(a.rel_path || '', b.rel_path || '', sortCfg.dir);
                                        }
                                      })
                                      .map((c) => (
                                      <tr key={`${h.version}-${c.rel_path}`}>
                                        <td>
                                          <span className={`diff-badge ${c.change}`}>{c.change}</span>
                                        </td>
                                        <td>
                                          <span className={`type-badge ${c.is_dir ? 'dir' : 'file'}`}>
                                            {c.is_dir ? 'DIR' : 'FILE'}
                                          </span>
                                        </td>
                                        <td className="mono-cell">{c.rel_path}</td>
                                        <td>
                                          <span className="hash-chip">
                                            {c.change === 'modified'
                                              ? `${c.base_hash || '—'} → ${c.target_hash || '—'}`
                                              : c.target_hash || c.base_hash || '—'}
                                          </span>
                                        </td>
                                        <td className="mono-cell">
                                          {c.change === 'added'
                                            ? formatBytes(c.target_size)
                                            : c.change === 'deleted'
                                              ? formatBytes(c.base_size)
                                              : `${formatBytes(c.base_size)} → ${formatBytes(c.target_size)}`}
                                        </td>
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </div>
                            )}
                          </div>
                        ))
                      )}
                    </div>
                  ) : (
                    <div className="table-wrapper">
                      <table className="data-table" id="scan-entries-table">
                        <thead>
                          <tr>
                            {renderSortableTh('scanEntries', 'is_dir', 'Type', 'asc')}
                            {renderSortableTh('scanEntries', 'rel_path', 'Relative Path', 'asc')}
                            {renderSortableTh('scanEntries', 'hash', `Hash (${selectedScan.hash_algo})`, 'asc')}
                            {renderSortableTh('scanEntries', 'size', 'Size', 'desc')}
                            {renderSortableTh('scanEntries', 'mode', 'Mode', 'asc')}
                            {renderSortableTh('scanEntries', 'owner', 'Owner', 'asc')}
                          </tr>
                        </thead>
                        <tbody>
                          {filteredEntries.map((entry) => (
                            <tr key={entry.rel_path}>
                              <td>
                                <span className={`type-badge ${entry.is_dir ? 'dir' : 'file'}`}>
                                  {entry.is_dir ? 'DIR' : 'FILE'}
                                </span>
                              </td>
                              <td className="mono-cell">{entry.rel_path}</td>
                              <td>
                                <span className="hash-chip">{entry.hash || '—'}</span>
                              </td>
                              <td className="mono-cell">{formatBytes(entry.size)}</td>
                              <td className="mono-cell">{entry.mode}</td>
                              <td>
                                {entry.owner}:{entry.group}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )
                ) : (
                  <div className="empty-state">Select a directory card on the left to inspect its tree and history.</div>
                )}
              </section>
            </div>
          </div>
        )}

        {activeTab === 'diff' && (
          <div className={`op-workspace-split ${opDrawerOpen ? '' : 'drawer-collapsed'}`} id="diff-workspace">
            {opDrawerOpen && (
              <aside className="op-sidebar-panel" aria-label="Snapshot Comparison Selector">
                <div className="op-sidebar-header">
                  <div>
                    <div className="op-sidebar-title">Compare Snapshots</div>
                    <div className="op-sidebar-subtitle">Select base &amp; target snapshots</div>
                  </div>
                  <div className="toolbar-group">
                    <button
                      id="swap-diff-btn"
                      type="button"
                      className="btn"
                      onClick={() => {
                        const prevBase = baseScanId;
                        setBaseScanId(targetScanId);
                        setTargetScanId(prevBase);
                      }}
                      disabled={!baseScanId || !targetScanId || loadingDiff}
                      title="Swap Base and Target snapshots"
                    >
                      ⇄ Swap
                    </button>
                    <button
                      id="run-diff-btn"
                      type="button"
                      className="btn btn-primary"
                      onClick={handleRunDiff}
                      disabled={!baseScanId || !targetScanId || loadingDiff}
                    >
                      {loadingDiff ? '...' : 'Diff'}
                    </button>
                  </div>
                </div>

                <div className="op-sidebar-subfilter" style={{ display: 'flex', flexDirection: 'column', gap: '0.45rem' }}>
                  <select
                    id="base-scan-select"
                    className="op-compact-select full"
                    value={baseScanId}
                    onChange={(e) => setBaseScanId(e.target.value)}
                  >
                    <option value="">Base Scan...</option>
                    {scans.map((s) => (
                      <option key={s.id} value={s.id}>
                        Base: {s.hostname}:{s.root_path} ({s.id})
                      </option>
                    ))}
                  </select>
                  <select
                    id="target-scan-select"
                    className="op-compact-select full"
                    value={targetScanId}
                    onChange={(e) => setTargetScanId(e.target.value)}
                  >
                    <option value="">Target Scan...</option>
                    {scans.map((s) => (
                      <option key={s.id} value={s.id}>
                        Target: {s.hostname}:{s.root_path} ({s.id})
                      </option>
                    ))}
                  </select>
                </div>

                <div className="op-card-list">
                  {scans.map((s, idx) => {
                    const isBase = baseScanId === s.id;
                    const isTarget = targetScanId === s.id;
                    const pctVol =
                      totalBytesAcrossScans > 0
                        ? Math.round(((s.total_bytes || 0) / totalBytesAcrossScans) * 100)
                        : 0;
                    return renderOpCard({
                      id: `diff-op-card-${idx}`,
                      cardKey: s.id,
                      selected: isBase || isTarget,
                      badgeTone: isTarget ? 'red' : isBase ? 'blue' : 'dark',
                      title: s.root_path,
                      versionBadge: isBase && isTarget ? 'BASE=TGT' : isBase ? 'BASE' : isTarget ? 'TARGET' : `v${s.version || 1}`,
                      pct: pctVol,
                      leftTitle: s.hostname || 'Host',
                      leftStatus: isBase ? 'Baseline' : isTarget ? 'Target' : 'Available',
                      leftStatusTone: 'ok',
                      rightTitle: s.id,
                      rightDelta: formatBytes(s.total_bytes),
                      pills: [
                        `◫ ${s.total_files || 0}f`,
                        `#${(s.root_hash || '').slice(0, 6)}`,
                      ],
                      shareBadge: `${pctVol}%`,
                      onClick: () => {
                        if (!baseScanId || (baseScanId && targetScanId && baseScanId !== s.id)) {
                          setBaseScanId(s.id);
                        } else {
                          setTargetScanId(s.id);
                        }
                      },
                    });
                  })}
                </div>
              </aside>
            )}

            <div className="op-main-stage">
              {diffReport ? (
                <div className="surface-panel" style={{ flex: 1 }}>
                  <div className="panel-header">
                    <h2 className="panel-title">
                      Comparison: {diffReport.base_scan?.id} → {diffReport.target_scan?.id}
                    </h2>
                    <div className="toolbar-group">
                      {diffChangeFilter !== 'all' && (
                        <button
                          type="button"
                          className="btn"
                          onClick={() => setDiffChangeFilter('all')}
                        >
                          Show All ({(diffReport.changes || []).length})
                        </button>
                      )}
                      <span className="diff-badge added">+{diffReport.added} Added</span>
                      <span className="diff-badge modified">~{diffReport.modified} Modified</span>
                      <span className="diff-badge deleted">-{diffReport.deleted} Deleted</span>
                      <span className="tag-badge">={diffReport.unchanged} Unchanged</span>
                    </div>
                  </div>

                  {diffReport.identical ? (
                    <div className="empty-state">
                      Both snapshots are 100% identical (Root Hash: {diffReport.base_scan?.root_hash}).
                    </div>
                  ) : (
                    <div className="table-wrapper">
                      <table className="data-table" id="diff-results-table">
                        <thead>
                          <tr>
                            {renderSortableTh('diff', 'change', 'Status', 'asc')}
                            {renderSortableTh('diff', 'is_dir', 'Type', 'asc')}
                            {renderSortableTh('diff', 'rel_path', 'Relative Path', 'asc')}
                            {renderSortableTh('diff', 'base_hash', 'Base Hash', 'asc')}
                            {renderSortableTh('diff', 'target_hash', 'Target Hash', 'asc')}
                            {renderSortableTh('diff', 'size', 'Size Change', 'desc')}
                          </tr>
                        </thead>
                        <tbody>
                          {filteredDiffChanges.map((c) => (
                            <tr key={c.rel_path}>
                              <td>
                                <span className={`diff-badge ${c.change}`}>{c.change}</span>
                              </td>
                              <td>
                                <span className={`type-badge ${c.is_dir ? 'dir' : 'file'}`}>
                                  {c.is_dir ? 'DIR' : 'FILE'}
                                </span>
                              </td>
                              <td className="mono-cell">{c.rel_path}</td>
                              <td>
                                <span className="hash-chip">{c.base_hash || '—'}</span>
                              </td>
                              <td>
                                <span className="hash-chip">{c.target_hash || '—'}</span>
                              </td>
                              <td className="mono-cell">
                                {c.change === 'added'
                                  ? formatBytes(c.target_size)
                                  : c.change === 'deleted'
                                    ? formatBytes(c.base_size)
                                    : `${formatBytes(c.base_size)} → ${formatBytes(c.target_size)}`}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              ) : (
                <div className="surface-panel empty-state" style={{ flex: 1 }}>
                  <p>Select a Base and Target snapshot in the left panel and click Diff to compare directory states.</p>
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === 'duplicates' && (
          <div className={`op-workspace-split ${opDrawerOpen ? '' : 'drawer-collapsed'}`} id="duplicates-workspace">
            {opDrawerOpen && (
              <aside className="op-sidebar-panel" aria-label="Reported Duplicate Groups">
                <div className="op-sidebar-header">
                  <div>
                    <div className="op-sidebar-title">Reported Duplicates</div>
                    <div className="op-sidebar-subtitle">
                      {filteredDuplicates.length} group(s) · {formatBytes(totalWastedBytes)} wasted
                    </div>
                  </div>
                  <select
                    id="dup-scan-select"
                    className="op-compact-select"
                    value={dupScanFilter}
                    onChange={(e) => {
                      setDupScanFilter(e.target.value);
                      setSelectedDupKey('');
                    }}
                  >
                    <option value="">All Scans</option>
                    {scans.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.hostname}:{s.root_path}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="op-card-list">
                  {filteredDuplicates.map((group, idx) => {
                    const cardKey = `${group.hash}-${group.size}`;
                    const isSelected = selectedDupKey ? selectedDupKey === cardKey : idx === 0;
                    const firstFile = group.files?.[0];
                    const lastFile = group.files?.[group.files.length - 1];
                    const pctWaste =
                      totalWastedBytes > 0
                        ? Math.round(((group.wasted_bytes || 0) / totalWastedBytes) * 100)
                        : 50;
                    return renderOpCard({
                      id: `dup-op-card-${idx}`,
                      cardKey,
                      selected: isSelected,
                      badgeTone: idx === 0 ? 'red' : isSelected ? 'blue' : 'dark',
                      title: firstFile?.name || group.hash.slice(0, 12),
                      versionBadge: `${group.count}x`,
                      pct: pctWaste,
                      leftTitle: firstFile?.hostname || 'Host A',
                      leftStatus: 'Duplicate',
                      leftStatusTone: 'ok',
                      rightTitle: lastFile?.hostname || 'Host B',
                      rightDelta: `+${formatBytes(group.wasted_bytes)}`,
                      pills: [
                        `⛁ ${formatBytes(group.size)} ea`,
                        `#${group.hash.slice(0, 8)}`,
                      ],
                      shareBadge: `${pctWaste}%`,
                      onClick: () => {
                        setSelectedDupKey(cardKey);
                        const rowEl = document.getElementById(`dup-row-${cardKey}`);
                        if (rowEl) {
                          rowEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                        }
                      },
                    });
                  })}
                </div>
              </aside>
            )}

            <div className="op-main-stage">
              <div className="surface-panel" style={{ flex: 1 }}>
                <div className="panel-header">
                  <h2 className="panel-title">Duplicate File Groups ({filteredDuplicates.length})</h2>
                  <span className="tag-badge">
                    Wasted Volume: {formatBytes(totalWastedBytes)}
                  </span>
                </div>

                {filteredDuplicates.length === 0 ? (
                  <div className="empty-state">
                    No duplicate non-empty files detected in the selected scope.
                  </div>
                ) : (
                  <div className="table-wrapper">
                    <table className="data-table" id="duplicates-table">
                      <thead>
                        <tr>
                          {renderSortableTh('dups', 'hash', 'Content Hash', 'asc')}
                          {renderSortableTh('dups', 'size', 'File Size', 'desc')}
                          {renderSortableTh('dups', 'count', 'Copies', 'desc')}
                          {renderSortableTh('dups', 'wasted_bytes', 'Wasted Space', 'desc')}
                          {renderSortableTh('dups', 'location', 'Locations & Actions', 'asc')}
                        </tr>
                      </thead>
                      <tbody>
                        {filteredDuplicates.map((group, idx) => {
                          const rowKey = `${group.hash}-${group.size}`;
                          const isSelected = selectedDupKey ? selectedDupKey === rowKey : idx === 0;
                          return (
                            <tr
                              key={rowKey}
                              id={`dup-row-${rowKey}`}
                              className={`tree-row ${isSelected ? 'focused' : ''}`}
                              onClick={() => setSelectedDupKey(rowKey)}
                            >
                              <td>
                                <button
                                  type="button"
                                  className="hash-chip hash-copy-btn"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleCopyText(group.hash, `dup-hash-${rowKey}`);
                                  }}
                                  title="Click to copy content hash"
                                >
                                  {copiedKey === `dup-hash-${rowKey}` ? 'Copied ✓' : group.hash}
                                </button>
                              </td>
                              <td className="mono-cell">{formatBytes(group.size)}</td>
                              <td className="mono-cell">{group.count}</td>
                              <td className="mono-cell">{formatBytes(group.wasted_bytes)}</td>
                              <td className="mono-cell">
                                <div className="dup-location-list">
                                  {(group.files || []).map((f, fIdx) => {
                                    const locKey = `${f.scan_id}-${f.rel_path}-${fIdx}`;
                                    return (
                                      <div key={locKey} className="dup-location-row">
                                        <span className="dup-location-path">
                                          [{f.scan_id}] <strong>{f.hostname}</strong>:{f.abs_path}
                                        </span>
                                        <span className="dup-location-actions">
                                          <button
                                            type="button"
                                            className="loc-action-btn"
                                            onClick={(e) => {
                                              e.stopPropagation();
                                              handleCopyText(f.abs_path, `loc-copy-${locKey}`);
                                            }}
                                            title="Copy file path"
                                          >
                                            {copiedKey === `loc-copy-${locKey}` ? 'Copied ✓' : 'Copy'}
                                          </button>
                                          <button
                                            type="button"
                                            className="loc-action-btn primary"
                                            onClick={(e) => {
                                              e.stopPropagation();
                                              setSelectedScanId(f.scan_id);
                                              setDetailView('entries');
                                              setEntryFilter(f.name || f.rel_path || '');
                                              setActiveTab('scans');
                                            }}
                                            title="Inspect file in Computers & Scans"
                                          >
                                            Inspect →
                                          </button>
                                        </span>
                                      </div>
                                    );
                                  })}
                                </div>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
        </main>
      </div>
    </div>
  );
}
