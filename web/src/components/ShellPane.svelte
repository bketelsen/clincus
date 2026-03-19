<script lang="ts">
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import { connectShell } from '../lib/ws';
  import { onMount } from 'svelte';

  let { containerId, visible = true }: { containerId: string; visible?: boolean } = $props();

  let termDiv: HTMLDivElement;
  let fitAddon: FitAddon;
  let exited = $state(false);
  let conn: ReturnType<typeof connectShell> | null = null;
  let term: Terminal | null = null;

  $effect(() => {
    if (visible && fitAddon) {
      setTimeout(() => fitAddon.fit(), 50);
    }
  });

  function initShell() {
    exited = false;
    term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      theme: { background: '#1a1a2e', foreground: '#eee' },
    });
    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(termDiv);
    fitAddon.fit();

    conn = connectShell(
      containerId,
      (data) => term!.write(data),
      (_code) => {
        term!.write('\r\n[Shell exited - click to restart]\r\n');
        exited = true;
      },
      (msg) => term!.write(`\r\n[Error: ${msg}]\r\n`),
    );

    term.onData((data) => conn!.send({ type: 'input', data }));
    term.onResize(({ cols, rows }) => conn!.send({ type: 'resize', cols, rows }));

    setTimeout(() => {
      fitAddon.fit();
      conn!.send({ type: 'resize', cols: term!.cols, rows: term!.rows });
    }, 100);
  }

  function restart() {
    cleanup();
    initShell();
  }

  function cleanup() {
    if (conn) conn.close();
    if (term) term.dispose();
    conn = null;
    term = null;
  }

  onMount(() => {
    initShell();

    const onResize = () => {
      if (fitAddon) fitAddon.fit();
    };
    window.addEventListener('resize', onResize);

    return () => {
      window.removeEventListener('resize', onResize);
      cleanup();
    };
  });
</script>

{#if exited}
  <div class="exit-overlay">
    <button class="restart-btn" onclick={restart}>Restart Shell</button>
  </div>
{/if}
<div class="terminal-container" bind:this={termDiv}></div>

<style>
  .terminal-container { width: 100%; height: 100%; }
  :global(.xterm) { height: 100%; }
  .exit-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 10;
    display: flex;
    justify-content: center;
    padding: 12px;
    background: rgba(15, 15, 26, 0.9);
  }
  .restart-btn {
    padding: 6px 16px;
    background: #2a2a4a;
    color: #ccc;
    border: 1px solid #444;
    border-radius: 4px;
    cursor: pointer;
    font-family: inherit;
    font-size: 12px;
  }
  .restart-btn:hover {
    background: #3a3a5a;
    color: #eee;
  }
</style>
