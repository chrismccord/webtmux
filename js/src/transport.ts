/**
 * Transport interface - common abstraction for WebSocket and WebTransport.
 * Both transport implementations must implement this interface.
 */
export interface Transport {
    /**
     * Open the transport connection.
     * For WebSocket, this is a no-op since connection happens in constructor.
     * For WebTransport, this initiates the connection asynchronously.
     */
    open(): void;

    /**
     * Close the transport connection.
     */
    close(): void;

    /**
     * Send data over the transport.
     * @param data The string data to send
     */
    send(data: string): void;

    /**
     * Check if the transport is open or connecting.
     */
    isOpen(): boolean;

    /**
     * Set callback for when connection is opened.
     */
    onOpen(callback: () => void): void;

    /**
     * Set callback for when data is received.
     */
    onReceive(callback: (data: string) => void): void;

    /**
     * Set callback for when connection is closed.
     */
    onClose(callback: () => void): void;
}

/**
 * TransportFactory creates Transport instances.
 * Used for reconnection logic.
 */
export interface TransportFactory {
    create(): Transport;
    protocol(): string;
}

/**
 * Check if WebTransport is supported in the current browser.
 */
export function isWebTransportSupported(): boolean {
    return typeof WebTransport !== 'undefined';
}
