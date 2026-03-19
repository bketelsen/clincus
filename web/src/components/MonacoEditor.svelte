<script lang="ts">
  import { onMount } from 'svelte';
  import * as monaco from 'monaco-editor';
  import { api } from '../lib/api';

  // Configure Monaco workers via import.meta.url (Vite native)
  import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
  import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
  import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker';
  import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';
  import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';

  self.MonacoEnvironment = {
    getWorker(_: string, label: string) {
      if (label === 'json') return new jsonWorker();
      if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker();
      if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker();
      if (label === 'typescript' || label === 'javascript') return new tsWorker();
      return new editorWorker();
    },
  };

  let { sessionId, filePath, visible = true }: {
    sessionId: string;
    filePath: string;
    visible?: boolean;
  } = $props();

  let editorDiv: HTMLDivElement;
  let editor: monaco.editor.IStandaloneCodeEditor;
  let dirty = $state(false);
  let saving = $state(false);
  let loadError = $state('');
  let currentPath = '';

  // Map of filePath -> model, so we preserve state per file
  const models = new Map<string, monaco.editor.ITextModel>();

  $effect(() => {
    if (visible && editor) {
      setTimeout(() => editor.layout(), 50);
    }
  });

  $effect(() => {
    if (filePath && editor && filePath !== currentPath) {
      loadFile(filePath);
    }
  });

  function getLanguageForPath(path: string): string {
    const ext = path.split('.').pop()?.toLowerCase() ?? '';
    const langMap: Record<string, string> = {
      go: 'go', ts: 'typescript', tsx: 'typescript', js: 'javascript', jsx: 'javascript',
      py: 'python', rs: 'rust', rb: 'ruby', java: 'java', c: 'c', cpp: 'cpp', h: 'c',
      cs: 'csharp', php: 'php', swift: 'swift', kt: 'kotlin',
      html: 'html', css: 'css', scss: 'scss', less: 'less',
      json: 'json', yaml: 'yaml', yml: 'yaml', toml: 'toml', xml: 'xml',
      md: 'markdown', sh: 'shell', bash: 'shell', zsh: 'shell',
      dockerfile: 'dockerfile', sql: 'sql', graphql: 'graphql',
      svelte: 'html', vue: 'html',
    };
    return langMap[ext] || 'plaintext';
  }

  async function loadFile(path: string) {
    loadError = '';
    dirty = false;
    currentPath = path;

    // Reuse existing model if we've already loaded this file
    if (models.has(path)) {
      editor.setModel(models.get(path)!);
      return;
    }

    try {
      const res = await api.readFile(sessionId, path);
      const lang = getLanguageForPath(path);
      const uri = monaco.Uri.parse(`file:///${path}`);
      const model = monaco.editor.createModel(res.content, lang, uri);
      model.onDidChangeContent(() => { dirty = true; });
      models.set(path, model);
      editor.setModel(model);
    } catch (e: any) {
      loadError = e?.message || 'Failed to load file';
    }
  }

  async function save() {
    if (!currentPath || saving) return;
    saving = true;
    try {
      const content = editor.getValue();
      await api.writeFile(sessionId, currentPath, content);
      dirty = false;
    } catch (e: any) {
      loadError = e?.message || 'Failed to save';
    }
    saving = false;
  }

  onMount(() => {
    editor = monaco.editor.create(editorDiv, {
      theme: 'vs-dark',
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      minimap: { enabled: false },
      automaticLayout: false,
      scrollBeyondLastLine: false,
    });

    // Ctrl+S / Cmd+S to save
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => save());

    if (filePath) loadFile(filePath);

    const onResize = () => editor.layout();
    window.addEventListener('resize', onResize);

    return () => {
      window.removeEventListener('resize', onResize);
      // Dispose all models
      for (const model of models.values()) model.dispose();
      models.clear();
      editor.dispose();
    };
  });
</script>

<div class="monaco-wrapper">
  {#if currentPath}
    <div class="editor-status">
      <span class="editor-filename">
        {currentPath.split('/').pop()}
        {#if dirty}<span class="dirty-dot" title="Unsaved changes"></span>{/if}
      </span>
      {#if saving}
        <span class="save-status">Saving...</span>
      {/if}
      {#if loadError}
        <span class="load-error">{loadError}</span>
      {/if}
    </div>
  {:else}
    <div class="no-file">Select a file to edit</div>
  {/if}
  <div class="editor-container" bind:this={editorDiv}></div>
</div>

<style>
  .monaco-wrapper {
    display: flex;
    flex-direction: column;
    height: 100%;
    flex: 1;
  }
  .editor-status {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 12px;
    background: #1e1e30;
    border-bottom: 1px solid #333;
    font-size: 12px;
  }
  .editor-filename { color: #ccc; }
  .dirty-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    background: #e8a838;
    border-radius: 50%;
    margin-left: 4px;
    vertical-align: middle;
  }
  .save-status { color: #888; font-size: 11px; }
  .load-error { color: #e66; font-size: 11px; }
  .no-file {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #555;
    font-size: 14px;
  }
  .editor-container { flex: 1; }
</style>
