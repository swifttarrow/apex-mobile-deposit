import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    lib: {
      entry: path.resolve(__dirname, 'src/entry.jsx'),
      name: 'FlowDiagram',
      fileName: 'flow-diagram',
      formats: ['iife'],
    },
    outDir: path.resolve(__dirname, '../cmd/server/web/scenarios'),
    emptyOutDir: false,
    rollupOptions: {
      output: {
        entryFileNames: 'flow-diagram.js',
      },
    },
  },
});
