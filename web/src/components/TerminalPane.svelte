<script lang="ts">
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import { connectTerminal } from '../lib/ws';
  import { onMount } from 'svelte';

  let { containerId, visible = true }: { containerId: string; visible?: boolean } = $props();

  let termDiv: HTMLDivElement;
  let fitAddon: FitAddon;

  $effect(() => {
    if (visible && fitAddon) {
      // Small delay to allow DOM to update display before measuring
      setTimeout(() => fitAddon.fit(), 50);
    }
  });

  onMount(() => {
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      theme: { background: '#1a1a2e', foreground: '#eee' },
    });
    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(termDiv);
    fitAddon.fit();

    const conn = connectTerminal(
      containerId,
      (data) => term.write(data),
      (code) => term.write(`\r\n[Process exited with code ${code}]\r\n`),
      (msg) => term.write(`\r\n[Error: ${msg}]\r\n`),
    );

    term.onData((data) => conn.send({ type: 'input', data }));
    term.onResize(({ cols, rows }) => conn.send({ type: 'resize', cols, rows }));

    setTimeout(() => {
      fitAddon.fit();
      conn.send({ type: 'resize', cols: term.cols, rows: term.rows });
    }, 100);

    const onResize = () => fitAddon.fit();
    window.addEventListener('resize', onResize);

    return () => {
      window.removeEventListener('resize', onResize);
      conn.close();
      term.dispose();
    };
  });
</script>

<div class="terminal-container" bind:this={termDiv}></div>

<style>
  .terminal-container { width: 100%; height: 100%; }
  :global(.xterm) { height: 100%; }
</style>
