import type { Transport, TransportFactory } from './transport';

/**
 * WebTransport connection wrapper that implements the Transport interface.
 * Uses length-prefixed framing to match WebSocket message semantics.
 */
export class WebTransportConnection implements Transport {
    private url: string;
    private transport: WebTransport | null = null;
    private stream: WritableStreamDefaultWriter<Uint8Array> | null = null;
    private reader: ReadableStreamDefaultReader<Uint8Array> | null = null;

    private openCallback: (() => void) | null = null;
    private receiveCallback: ((data: string) => void) | null = null;
    private closeCallback: (() => void) | null = null;

    private isConnected: boolean = false;
    private readBuffer: Uint8Array = new Uint8Array(0);

    constructor(url: string) {
        this.url = url;
    }

    open(): void {
        this.connect();
    }

    private async connect(): Promise<void> {
        try {
            this.transport = new WebTransport(this.url);

            // Wait for connection to be ready
            await this.transport.ready;
            this.isConnected = true;
            console.log('WebTransport connection ready');

            // Open bidirectional stream
            const stream = await this.transport.createBidirectionalStream();
            this.stream = stream.writable.getWriter();
            this.reader = stream.readable.getReader();

            // Start reading loop
            this.startReading();

            // Call open callback
            if (this.openCallback) {
                this.openCallback();
            }

            // Handle connection close
            this.transport.closed.then(() => {
                console.log('WebTransport connection closed');
                this.isConnected = false;
                if (this.closeCallback) {
                    this.closeCallback();
                }
            }).catch((error) => {
                console.error('WebTransport closed with error:', error);
                this.isConnected = false;
                if (this.closeCallback) {
                    this.closeCallback();
                }
            });

        } catch (error) {
            console.error('WebTransport connection failed:', error);
            this.isConnected = false;
            if (this.closeCallback) {
                this.closeCallback();
            }
        }
    }

    close(): void {
        this.isConnected = false;
        if (this.reader) {
            this.reader.cancel().catch(() => {});
        }
        if (this.stream) {
            this.stream.close().catch(() => {});
        }
        if (this.transport) {
            this.transport.close();
        }
    }

    send(data: string): void {
        if (!this.stream || !this.isConnected) {
            console.error('WebTransport not connected, cannot send');
            return;
        }

        // Encode string to bytes
        const encoder = new TextEncoder();
        const payload = encoder.encode(data);

        // Create length-prefixed frame (2 bytes big-endian length + payload)
        const frame = new Uint8Array(2 + payload.length);
        frame[0] = (payload.length >> 8) & 0xff;
        frame[1] = payload.length & 0xff;
        frame.set(payload, 2);

        this.stream.write(frame).catch((error) => {
            console.error('WebTransport send error:', error);
        });
    }

    isOpen(): boolean {
        return this.isConnected;
    }

    onOpen(callback: () => void): void {
        this.openCallback = callback;
    }

    onReceive(callback: (data: string) => void): void {
        this.receiveCallback = callback;
    }

    onClose(callback: () => void): void {
        this.closeCallback = callback;
    }

    private async startReading(): Promise<void> {
        if (!this.reader) return;

        const decoder = new TextDecoder();

        try {
            while (this.isConnected) {
                const { value, done } = await this.reader.read();

                if (done) {
                    console.log('WebTransport stream ended');
                    break;
                }

                if (!value) continue;

                // Append to read buffer
                const newBuffer = new Uint8Array(this.readBuffer.length + value.length);
                newBuffer.set(this.readBuffer);
                newBuffer.set(value, this.readBuffer.length);
                this.readBuffer = newBuffer;

                // Process complete frames
                while (this.readBuffer.length >= 2) {
                    // Read length prefix (2 bytes big-endian)
                    const length = (this.readBuffer[0] << 8) | this.readBuffer[1];

                    // Check if we have complete frame
                    if (this.readBuffer.length < 2 + length) {
                        break; // Wait for more data
                    }

                    // Extract payload
                    const payload = this.readBuffer.slice(2, 2 + length);
                    this.readBuffer = this.readBuffer.slice(2 + length);

                    // Decode and deliver message
                    const message = decoder.decode(payload);
                    if (this.receiveCallback) {
                        this.receiveCallback(message);
                    }
                }
            }
        } catch (error) {
            if (this.isConnected) {
                console.error('WebTransport read error:', error);
            }
        }
    }
}

/**
 * Factory for creating WebTransport connections.
 */
export class WebTransportFactory implements TransportFactory {
    private url: string;

    constructor(url: string) {
        this.url = url;
    }

    create(): Transport {
        return new WebTransportConnection(this.url);
    }

    protocol(): string {
        return 'webtransport';
    }
}
