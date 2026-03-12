import { mount } from './flow-diagram.jsx';

function init() {
  const el = document.getElementById('flow-diagram-root');
  if (el) {
    const api = mount(el);
    window.__flowDiagram = api;
  }
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}
