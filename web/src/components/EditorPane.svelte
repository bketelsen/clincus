<script lang="ts">
  import FileTree from './FileTree.svelte';
  import MonacoEditor from './MonacoEditor.svelte';

  let { sessionId, visible = true }: { sessionId: string; visible?: boolean } = $props();

  let selectedFile = $state('');

  function onFileSelect(path: string) {
    selectedFile = path;
  }
</script>

<div class="editor-pane">
  <FileTree {sessionId} {onFileSelect} />
  <MonacoEditor {sessionId} filePath={selectedFile} {visible} />
</div>

<style>
  .editor-pane {
    display: flex;
    height: 100%;
  }
</style>
