import React, { useEffect, useState, useCallback } from 'react';
import { createRoot } from 'react-dom/client';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

const nodeStyle = {
  background: '#161b22',
  border: '1px solid #30363d',
  borderRadius: 6,
  color: '#e6edf3',
  fontSize: 11,
  fontFamily: 'JetBrains Mono, monospace',
};

const initialNodes = [
  { id: 'start', type: 'input', position: { x: 200, y: 0 }, data: { label: 'Start' }, style: { ...nodeStyle, width: 60 } },
  { id: 'Requested', position: { x: 180, y: 80 }, data: { label: 'Requested' }, style: nodeStyle },
  { id: 'Validating', position: { x: 180, y: 160 }, data: { label: 'Validating' }, style: nodeStyle },
  { id: 'ManualReview', position: { x: 360, y: 200 }, data: { label: 'ManualReview' }, style: nodeStyle },
  { id: 'Analyzing', position: { x: 180, y: 260 }, data: { label: 'Analyzing' }, style: nodeStyle },
  { id: 'Rejected', position: { x: 400, y: 340 }, data: { label: 'Rejected' }, style: { ...nodeStyle, borderColor: '#f85149' } },
  { id: 'Approved', position: { x: 180, y: 360 }, data: { label: 'Approved' }, style: nodeStyle },
  { id: 'FundsPosted', position: { x: 180, y: 440 }, data: { label: 'FundsPosted' }, style: nodeStyle },
  { id: 'SettlementQueued', position: { x: 180, y: 520 }, data: { label: 'SettlementQueued' }, style: nodeStyle },
  { id: 'Settling', position: { x: 180, y: 600 }, data: { label: 'Settling' }, style: nodeStyle },
  { id: 'Completed', position: { x: 80, y: 680 }, data: { label: 'Completed' }, style: { ...nodeStyle, borderColor: '#3fb950' } },
  { id: 'Returned', position: { x: 280, y: 680 }, data: { label: 'Returned' }, style: { ...nodeStyle, borderColor: '#f85149' } },
  { id: 'SettlementIssue', position: { x: 360, y: 640 }, data: { label: 'SettlementIssue' }, style: { ...nodeStyle, borderColor: '#f85149' } },
];

const edgeDefaults = { type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, color: '#484f58' }, style: { stroke: '#484f58', strokeWidth: 2 } };

const initialEdges = [
  { id: 'start-Requested', source: 'start', target: 'Requested', label: 'Submit', ...edgeDefaults },
  { id: 'Requested-Validating', source: 'Requested', target: 'Validating', label: 'Send to Vendor', ...edgeDefaults },
  { id: 'Validating-Requested', source: 'Validating', target: 'Requested', label: 'IQA Fail', ...edgeDefaults },
  { id: 'Validating-Rejected', source: 'Validating', target: 'Rejected', label: 'Duplicate', ...edgeDefaults },
  { id: 'Validating-ManualReview', source: 'Validating', target: 'ManualReview', label: 'MICR/Amount', ...edgeDefaults },
  { id: 'Validating-Analyzing', source: 'Validating', target: 'Analyzing', label: 'Clean pass', ...edgeDefaults },
  { id: 'ManualReview-Rejected', source: 'ManualReview', target: 'Rejected', label: 'Reject', ...edgeDefaults },
  { id: 'ManualReview-Analyzing', source: 'ManualReview', target: 'Analyzing', label: 'Approve', ...edgeDefaults },
  { id: 'Analyzing-Rejected', source: 'Analyzing', target: 'Rejected', label: 'Rules fail', ...edgeDefaults },
  { id: 'Analyzing-Approved', source: 'Analyzing', target: 'Approved', label: 'Pass', ...edgeDefaults },
  { id: 'Approved-FundsPosted', source: 'Approved', target: 'FundsPosted', label: 'Ledger', ...edgeDefaults },
  { id: 'FundsPosted-SettlementQueued', source: 'FundsPosted', target: 'SettlementQueued', label: 'EOD batch', ...edgeDefaults },
  { id: 'FundsPosted-Returned', source: 'FundsPosted', target: 'Returned', label: 'Return', ...edgeDefaults },
  { id: 'SettlementQueued-Settling', source: 'SettlementQueued', target: 'Settling', label: 'Submit', ...edgeDefaults },
  { id: 'Settling-Completed', source: 'Settling', target: 'Completed', label: 'Ack', ...edgeDefaults },
  { id: 'Settling-SettlementIssue', source: 'Settling', target: 'SettlementIssue', label: 'Missing ack', ...edgeDefaults },
  { id: 'Completed-Returned', source: 'Completed', target: 'Returned', label: 'Return + fee', ...edgeDefaults },
];

function edgeId(from, to) {
  const norm = (s) => s === '[*]' ? 'start' : s;
  return `${norm(from)}-${norm(to)}`;
}

function FlowDiagramInner({ onReady }) {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  const resetEdges = useCallback(() => {
    setEdges((eds) =>
      eds.map((e) => ({
        ...e,
        style: { ...edgeDefaults.style, stroke: '#484f58', strokeWidth: 2 },
        className: '',
      }))
    );
  }, [setEdges]);

  const animatePath = useCallback(
    async (segments, isFail) => {
      if (!segments?.length) return;
      const color = isFail ? '#f85149' : '#3fb950';
      for (let i = 0; i < segments.length; i++) {
        const [from, to] = segments[i];
        const id = edgeId(from, to);
        setEdges((eds) =>
          eds.map((e) => {
            if (e.id === id) {
              return { ...e, style: { ...e.style, stroke: '#58a6ff', strokeWidth: 3 }, className: 'flow-edge-current' };
            }
            const segId = (f, t) => `${(f === '[*]' ? 'start' : f)}-${(t === '[*]' ? 'start' : t)}`;
            const isTraversed = segments.slice(0, i).some(([f, t]) => segId(f, t) === e.id);
            if (isTraversed) {
              return { ...e, style: { ...e.style, stroke: color, strokeWidth: 2.5 }, className: 'flow-edge-traversed' };
            }
            return { ...e, style: { ...edgeDefaults.style, stroke: '#484f58', strokeWidth: 2 }, className: '' };
          })
        );
        await new Promise((r) => setTimeout(r, 1000));
      }
      setEdges((eds) =>
        eds.map((e) => {
          const segId = (f, t) => `${(f === '[*]' ? 'start' : f)}-${(t === '[*]' ? 'start' : t)}`;
          const traversed = segments.some(([f, t]) => segId(f, t) === e.id);
          return traversed ? { ...e, style: { ...e.style, stroke: color, strokeWidth: 2.5 }, className: 'flow-edge-traversed' } : { ...e, style: { ...edgeDefaults.style, stroke: '#484f58', strokeWidth: 2 }, className: '' };
        })
      );
    },
    [setEdges]
  );

  useEffect(() => {
    onReady({ animatePath, resetEdges });
  }, [onReady, animatePath, resetEdges]);

  useEffect(() => {
    const handler = (e) => {
      const { segments, isFail } = e.detail || {};
      animatePath(segments, isFail);
    };
    const resetHandler = () => resetEdges();
    window.addEventListener('flow-animate', handler);
    window.addEventListener('flow-reset', resetHandler);
    return () => {
      window.removeEventListener('flow-animate', handler);
      window.removeEventListener('flow-reset', resetHandler);
    };
  }, [animatePath, resetEdges]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      proOptions={{ hideAttribution: true }}
      style={{ background: '#0d1117' }}
    >
      <Background color="#21262d" gap={12} />
      <Controls />
      <MiniMap nodeColor="#161b22" maskColor="rgba(0,0,0,0.6)" />
    </ReactFlow>
  );
}

export function mount(container) {
  if (!container) return null;
  const root = createRoot(container);
  root.render(
    <FlowDiagramInner
      onReady={({ animatePath, resetEdges }) => {
        window.__flowAnimate = animatePath;
        window.__flowReset = resetEdges;
      }}
    />
  );
  return {
    animatePath: (segments, isFail) => window.dispatchEvent(new CustomEvent('flow-animate', { detail: { segments, isFail } })),
    reset: () => window.dispatchEvent(new CustomEvent('flow-reset')),
  };
}
