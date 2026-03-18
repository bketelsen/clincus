import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { connectEvents } from './ws';

class MockWebSocket {
    static instances: MockWebSocket[] = [];
    onopen: (() => void) | null = null;
    onclose: (() => void) | null = null;
    onmessage: ((evt: any) => void) | null = null;
    onerror: (() => void) | null = null;
    readyState = 0;
    closeCalled = false;

    constructor(public url: string) {
        MockWebSocket.instances.push(this);
    }

    close() {
        this.closeCalled = true;
        this.readyState = 3;
    }
    send(_data: string) {}

    simulateOpen() {
        this.readyState = 1;
        this.onopen?.();
    }

    simulateClose() {
        this.onclose?.();
    }

    simulateMessage(data: any) {
        this.onmessage?.({ data: JSON.stringify(data) });
    }
}

// Need to also stub location for the WebSocket URL construction
const mockLocation = { protocol: 'http:', host: 'localhost:8080', hash: '' };

describe('connectEvents', () => {
    beforeEach(() => {
        MockWebSocket.instances = [];
        vi.stubGlobal('WebSocket', MockWebSocket);
        vi.stubGlobal('location', mockLocation);
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.restoreAllMocks();
        vi.useRealTimers();
    });

    it('creates a WebSocket connection on call', () => {
        connectEvents(() => {});
        expect(MockWebSocket.instances).toHaveLength(1);
        expect(MockWebSocket.instances[0].url).toBe('ws://localhost:8080/ws/events');
    });

    it('calls onReconnect only after new connection opens, not immediately', () => {
        const onReconnect = vi.fn();
        const { close } = connectEvents(() => {}, onReconnect);

        const firstWs = MockWebSocket.instances[0];
        firstWs.simulateOpen();

        // First onReconnect call is on initial connect -- that's expected
        onReconnect.mockClear();

        // Simulate disconnect
        firstWs.simulateClose();

        // Advance past reconnect delay
        vi.advanceTimersByTime(2000);

        // A new WebSocket should be created but onReconnect NOT yet called
        expect(MockWebSocket.instances).toHaveLength(2);
        expect(onReconnect).not.toHaveBeenCalled();

        // Now simulate the new connection opening
        MockWebSocket.instances[1].simulateOpen();
        expect(onReconnect).toHaveBeenCalledTimes(1);

        close();
    });

    it('closes old WebSocket before creating new one on reconnect', () => {
        const { close } = connectEvents(() => {});
        const firstWs = MockWebSocket.instances[0];
        firstWs.simulateOpen();

        // Simulate disconnect
        firstWs.simulateClose();
        vi.advanceTimersByTime(2000);

        // Old WS should have been closed and onclose nulled
        expect(firstWs.closeCalled).toBe(true);
        expect(firstWs.onclose).toBeNull();

        close();
    });

    it('uses exponential backoff for reconnection', () => {
        const { close } = connectEvents(() => {});

        // First connection opens then closes
        MockWebSocket.instances[0].simulateOpen();
        MockWebSocket.instances[0].simulateClose();

        // First reconnect after 2s
        vi.advanceTimersByTime(1999);
        expect(MockWebSocket.instances).toHaveLength(1);
        vi.advanceTimersByTime(1);
        expect(MockWebSocket.instances).toHaveLength(2);

        // Second disconnect, reconnect after 4s
        MockWebSocket.instances[1].simulateClose();
        vi.advanceTimersByTime(3999);
        expect(MockWebSocket.instances).toHaveLength(2);
        vi.advanceTimersByTime(1);
        expect(MockWebSocket.instances).toHaveLength(3);

        // Third disconnect, reconnect after 8s
        MockWebSocket.instances[2].simulateClose();
        vi.advanceTimersByTime(7999);
        expect(MockWebSocket.instances).toHaveLength(3);
        vi.advanceTimersByTime(1);
        expect(MockWebSocket.instances).toHaveLength(4);

        close();
    });

    it('resets backoff delay on successful connection', () => {
        const { close } = connectEvents(() => {});

        // Build up delay
        MockWebSocket.instances[0].simulateOpen();
        MockWebSocket.instances[0].simulateClose();
        vi.advanceTimersByTime(2000); // reconnect at 2s

        MockWebSocket.instances[1].simulateClose();
        vi.advanceTimersByTime(4000); // reconnect at 4s

        // Now open successfully -- should reset delay
        MockWebSocket.instances[2].simulateOpen();
        MockWebSocket.instances[2].simulateClose();

        // Should reconnect after 2s again (reset), not 8s
        vi.advanceTimersByTime(2000);
        expect(MockWebSocket.instances).toHaveLength(4);

        close();
    });

    it('forwards events to onEvent callback', () => {
        const onEvent = vi.fn();
        connectEvents(onEvent);

        MockWebSocket.instances[0].simulateOpen();
        MockWebSocket.instances[0].simulateMessage({ type: 'session.started', id: 'abc' });

        expect(onEvent).toHaveBeenCalledWith({ type: 'session.started', id: 'abc' });
    });

    it('close() prevents reconnection', () => {
        const { close } = connectEvents(() => {});
        MockWebSocket.instances[0].simulateOpen();

        close();

        // Even after simulating disconnect, no reconnect should happen
        vi.advanceTimersByTime(10000);
        expect(MockWebSocket.instances).toHaveLength(1);
    });
});
