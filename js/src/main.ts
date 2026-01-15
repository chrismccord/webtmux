import { isWebTransportSupported, TransportFactory } from "./transport";
import { WebSocketFactory, WebSocketConnection } from "./websocket";
import { WebTransportConnection } from "./webtransport";
import { Terminal, WebTTY, protocols } from "./webtty";
import { OurXterm } from "./xterm";

// Configuration variables injected by server
declare var gotty_auth_token: string;
declare var gotty_ws_query_args: string;
declare var gotty_webtransport_enabled: boolean;
declare var gotty_webtransport_port: string;

/**
 * FallbackTransportFactory attempts WebTransport first, falls back to WebSocket on failure.
 */
class FallbackTransportFactory implements TransportFactory {
    private wsUrl: string;
    private wtUrl: string;
    private protocols: string[];
    private wtFailed: boolean = false;
    private activeProtocol: string = 'webtransport';

    constructor(wsUrl: string, wtUrl: string, protocols: string[]) {
        this.wsUrl = wsUrl;
        this.wtUrl = wtUrl;
        this.protocols = protocols;
    }

    create(): WebSocketConnection | FallbackTransport {
        if (this.wtFailed) {
            this.activeProtocol = 'websocket';
            return new WebSocketConnection(this.wsUrl, this.protocols);
        }

        this.activeProtocol = 'webtransport';
        return new FallbackTransport(
            () => new WebTransportConnection(this.wtUrl),
            () => new WebSocketConnection(this.wsUrl, this.protocols),
            () => { this.wtFailed = true; }
        );
    }

    protocol(): string {
        return this.activeProtocol;
    }
}

/**
 * FallbackTransport wraps a transport and handles fallback on connection failure.
 */
class FallbackTransport {
    private createWT: () => WebTransportConnection;
    private createWS: () => WebSocketConnection;
    private onWTFailed: () => void;
    private activeTransport: WebTransportConnection | WebSocketConnection | null = null;
    private isEstablished: boolean = false;

    private callbacks: {
        open: (() => void) | null;
        receive: ((data: string) => void) | null;
        close: (() => void) | null;
    } = { open: null, receive: null, close: null };

    constructor(
        createWT: () => WebTransportConnection,
        createWS: () => WebSocketConnection,
        onWTFailed: () => void
    ) {
        this.createWT = createWT;
        this.createWS = createWS;
        this.onWTFailed = onWTFailed;
    }

    open(): void {
        // Try WebTransport first
        this.activeTransport = this.createWT();

        // Set up close handler to detect failure
        this.activeTransport.onClose(() => {
            if (!this.isEstablished) {
                console.log('WebTransport connection failed, falling back to WebSocket');
                this.onWTFailed();
                this.activeTransport = this.createWS();
                this.setupCallbacks();
                this.activeTransport.open();
            } else {
                if (this.callbacks.close) this.callbacks.close();
            }
        });

        // Set up open handler
        this.activeTransport.onOpen(() => {
            this.isEstablished = true;
            console.log('WebTransport connection established');
            if (this.callbacks.open) this.callbacks.open();
        });

        // Set up receive handler
        this.activeTransport.onReceive((data) => {
            this.isEstablished = true;
            if (this.callbacks.receive) this.callbacks.receive(data);
        });

        this.activeTransport.open();
    }

    private setupCallbacks(): void {
        if (!this.activeTransport) return;

        this.activeTransport.onOpen(() => {
            this.isEstablished = true;
            if (this.callbacks.open) this.callbacks.open();
        });
        this.activeTransport.onReceive((data) => {
            this.isEstablished = true;
            if (this.callbacks.receive) this.callbacks.receive(data);
        });
        this.activeTransport.onClose(() => {
            if (this.callbacks.close) this.callbacks.close();
        });
    }

    close(): void {
        this.activeTransport?.close();
    }

    send(data: string): void {
        this.activeTransport?.send(data);
    }

    isOpen(): boolean {
        return this.activeTransport?.isOpen() ?? false;
    }

    onOpen(callback: () => void): void {
        this.callbacks.open = callback;
    }

    onReceive(callback: (data: string) => void): void {
        this.callbacks.receive = callback;
    }

    onClose(callback: () => void): void {
        this.callbacks.close = callback;
    }
}

/**
 * Create the best available transport factory based on configuration.
 */
function createTransportFactory(
    wsUrl: string,
    wtUrl: string | null,
    protocols: string[]
): TransportFactory {
    if (wtUrl && isWebTransportSupported()) {
        console.log('WebTransport is supported, will try WebTransport first with WebSocket fallback');
        return new FallbackTransportFactory(wsUrl, wtUrl, protocols);
    }

    console.log('Using WebSocket transport only');
    return new WebSocketFactory(wsUrl, protocols);
}

const elem = document.getElementById("terminal");

if (elem !== null) {
    var term: Terminal;
    term = new OurXterm(elem);

    const httpsEnabled = window.location.protocol == "https:";
    const queryArgs = (gotty_ws_query_args === "") ? "" : "?" + gotty_ws_query_args;

    // WebSocket URL (always available)
    const wsUrl = (httpsEnabled ? 'wss://' : 'ws://') +
        window.location.host +
        window.location.pathname + 'ws' + queryArgs;

    // WebTransport URL (only if enabled and HTTPS)
    let wtUrl: string | null = null;
    if (httpsEnabled && gotty_webtransport_enabled && isWebTransportSupported()) {
        const wtHost = window.location.hostname + ':' + gotty_webtransport_port;
        wtUrl = 'https://' + wtHost + window.location.pathname + 'wt' + queryArgs;
        console.log('WebTransport URL configured:', wtUrl);
    }

    const args = window.location.search;

    // Create factory with automatic fallback
    const factory = createTransportFactory(wsUrl, wtUrl, protocols);
    console.log(`Initial transport protocol: ${factory.protocol()}`);

    const wt = new WebTTY(term, factory, args, gotty_auth_token);
    const closer = wt.open();

    window.addEventListener("unload", () => {
        closer();
        term.close();
    });
}
