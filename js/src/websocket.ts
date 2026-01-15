import type { Transport, TransportFactory } from './transport';

/**
 * WebSocket connection wrapper that implements the Transport interface.
 */
export class WebSocketConnection implements Transport {
    private bare: WebSocket;

    constructor(url: string, protocols: string[]) {
        this.bare = new WebSocket(url, protocols);
    }

    open(): void {
        // WebSocket connects automatically in constructor
    }

    close(): void {
        this.bare.close();
    }

    send(data: string): void {
        this.bare.send(data);
    }

    isOpen(): boolean {
        return this.bare.readyState === WebSocket.CONNECTING ||
               this.bare.readyState === WebSocket.OPEN;
    }

    onOpen(callback: () => void): void {
        this.bare.onopen = () => callback();
    }

    onReceive(callback: (data: string) => void): void {
        this.bare.onmessage = (event) => callback(event.data);
    }

    onClose(callback: () => void): void {
        this.bare.onclose = () => callback();
    }
}

/**
 * Factory for creating WebSocket connections.
 */
export class WebSocketFactory implements TransportFactory {
    private url: string;
    private protocols: string[];

    constructor(url: string, protocols: string[]) {
        this.url = url;
        this.protocols = protocols;
    }

    create(): Transport {
        return new WebSocketConnection(this.url, this.protocols);
    }

    protocol(): string {
        return 'websocket';
    }
}

// Backward compatibility exports
export { WebSocketFactory as ConnectionFactory };
export { WebSocketConnection as Connection };
