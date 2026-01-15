// WebTmux - Main entry point
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebglAddon } from '@xterm/addon-webgl';

// Import components
import './components/sidebar.js';
import './components/mobile-controls.js';
import './components/shortcuts.js';

// Protocol message types (must match Go constants)
const MSG = {
  // Input (client -> server)
  Input: '1',
  Ping: '2',
  ResizeTerminal: '3',
  SetEncoding: '4',
  TmuxSelectPane: '5',
  TmuxSelectWindow: '6',
  TmuxSplitPane: '7',
  TmuxClosePane: '8',
  TmuxCopyMode: '9',
  TmuxScrollUp: 'B',
  TmuxScrollDown: 'C',
  TmuxNewWindow: 'D',
  TmuxSwitchSession: 'E',

  // Output (server -> client)
  Output: '1',
  Pong: '2',
  SetWindowTitle: '3',
  SetPreferences: '4',
  SetReconnect: '5',
  SetBufferSize: '6',
  TmuxLayoutUpdate: '7',
  TmuxModeUpdate: '9',
};

class WebTmux {
  constructor() {
    this.terminal = null;
    this.fitAddon = null;
    this.ws = null;
    this.reconnectInterval = null;
    this.bufferSize = 1024 * 1024;
    this.inCopyMode = false;
    this.layout = null;
    this.pendingSessionSwitch = null;
    this.oscBuffer = ''; // Buffer for OSC sequence detection

    this.init();
  }

  init() {
    // Create terminal
    this.terminal = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1a1a2e',
        foreground: '#eaeaea',
        cursor: '#f0f0f0',
        selection: 'rgba(255, 255, 255, 0.3)',
      },
      scrollback: 0, // tmux handles scrollback via copy mode
      allowProposedApi: true,
    });

    // Add fit addon
    this.fitAddon = new FitAddon();
    this.terminal.loadAddon(this.fitAddon);

    // Open terminal
    const container = document.getElementById('terminal');
    this.terminal.open(container);

    // Try to load WebGL addon
    try {
      const webglAddon = new WebglAddon();
      this.terminal.loadAddon(webglAddon);
    } catch (e) {
      console.warn('WebGL addon not supported:', e);
    }

    // Fit terminal and focus
    this.fitAddon.fit();
    this.terminal.focus();

    // Setup resize observer
    const resizeObserver = new ResizeObserver(() => {
      this.fitAddon.fit();
      this.sendResize();
    });
    resizeObserver.observe(container);

    // Setup input handling
    this.encoder = new TextEncoder();

    // Intercept arrow keys and control chars to ensure correct sequences
    this.terminal.attachCustomKeyEventHandler((ev) => {
      // Only handle keydown events
      if (ev.type !== 'keydown') return true;

      // Allow Cmd+C / Ctrl+C to copy selected text
      if ((ev.metaKey || ev.ctrlKey) && ev.key === 'c') {
        const selection = this.terminal.getSelection();
        if (selection) {
          navigator.clipboard.writeText(selection).catch(err => {
            console.warn('Failed to copy:', err);
          });
          return false; // Handled
        }
        // No selection - let it pass through as Ctrl+C (interrupt)
        return true;
      }

      // Allow Cmd+V / Ctrl+V to paste
      if ((ev.metaKey || ev.ctrlKey) && ev.key === 'v') {
        ev.preventDefault(); // Prevent browser's native paste
        navigator.clipboard.readText().then(text => {
          if (text) {
            const bytes = this.encoder.encode(text);
            const binary = String.fromCharCode(...bytes);
            this.sendMessage(MSG.Input, btoa(binary));
          }
        }).catch(err => {
          console.warn('Failed to paste:', err);
        });
        return false; // Handled
      }

      // Map arrow keys to CSI sequences (ESC [ A/B/C/D)
      // Using CSI instead of SS3 for better compatibility
      const arrowMap = {
        'ArrowUp': '\x1b[A',
        'ArrowDown': '\x1b[B',
        'ArrowRight': '\x1b[C',
        'ArrowLeft': '\x1b[D',
      };

      if (arrowMap[ev.key]) {
        // Send raw CSI sequence
        const seq = arrowMap[ev.key];
        const binary = String.fromCharCode(...[...seq].map(c => c.charCodeAt(0)));
        this.sendMessage(MSG.Input, btoa(binary));
        return false; // Prevent xterm.js default handling
      }

      // Handle Ctrl+N (down) and Ctrl+P (up) for fzf navigation
      if (ev.ctrlKey && !ev.altKey && !ev.metaKey) {
        const ctrlMap = {
          'n': '\x0e', // Ctrl+N = 0x0e = 14
          'p': '\x10', // Ctrl+P = 0x10 = 16
          'j': '\x0a', // Ctrl+J = newline
          'k': '\x0b', // Ctrl+K
        };
        const key = ev.key.toLowerCase();
        if (ctrlMap[key]) {
          const binary = String.fromCharCode(ctrlMap[key].charCodeAt(0));
          this.sendMessage(MSG.Input, btoa(binary));
          return false;
        }
      }

      return true; // Let xterm.js handle other keys
    });

    this.terminal.onData((data) => {
      if (this.inCopyMode && data.length === 1) {
        // Exit copy mode on any key press (except scroll keys)
        this.sendMessage(MSG.TmuxCopyMode, '0');
        this.inCopyMode = false;
      }
      // Encode string to bytes, then to base64 (matches original gotty)
      const bytes = this.encoder.encode(data);
      const binary = String.fromCharCode(...bytes);
      this.sendMessage(MSG.Input, btoa(binary));
    });

    // Setup touch/scroll handling for copy mode
    this.setupTouchHandling();

    // Connect WebSocket
    this.connect();

    // Expose for components
    window.webtmux = this;
  }

  setupTouchHandling() {
    const container = document.getElementById('terminal');
    let touchStartY = 0;

    // Touch handling for mobile scroll -> copy mode
    container.addEventListener('touchstart', (e) => {
      touchStartY = e.touches[0].clientY;
    }, { passive: true });

    container.addEventListener('touchmove', (e) => {
      const deltaY = touchStartY - e.touches[0].clientY;
      const threshold = 30;

      if (Math.abs(deltaY) > threshold) {
        if (!this.inCopyMode) {
          this.sendMessage(MSG.TmuxCopyMode, '1');
          this.inCopyMode = true;
        }

        const lines = Math.floor(Math.abs(deltaY) / 20);
        if (lines > 0) {
          // Swipe up (deltaY > 0) = scroll DOWN in history (show newer)
          // Swipe down (deltaY < 0) = scroll UP in history (show older)
          if (deltaY > 0) {
            this.sendMessage(MSG.TmuxScrollDown, String(lines));
          } else {
            this.sendMessage(MSG.TmuxScrollUp, String(lines));
          }
          touchStartY = e.touches[0].clientY;
        }
      }
    }, { passive: true });

    // Mouse wheel for desktop scroll -> copy mode
    this.terminal.attachCustomWheelEventHandler((event) => {
      // Only intercept scroll up (entering history) - deltaY < 0 = wheel up
      if (event.deltaY < 0) {
        if (!this.inCopyMode) {
          this.sendMessage(MSG.TmuxCopyMode, '1');
          this.inCopyMode = true;
        }
      }

      if (this.inCopyMode) {
        const lines = Math.max(1, Math.floor(Math.abs(event.deltaY) / 50));
        // Wheel up (deltaY < 0) = scroll UP in tmux (show older history)
        // Wheel down (deltaY > 0) = scroll DOWN in tmux (show newer)
        if (event.deltaY < 0) {
          this.sendMessage(MSG.TmuxScrollUp, String(lines));
        } else {
          this.sendMessage(MSG.TmuxScrollDown, String(lines));
        }
        return false; // Prevent default scroll
      }

      return true; // Allow normal handling when not in copy mode
    });
  }

  connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}${window.location.pathname}ws`;

    this.ws = new WebSocket(wsUrl, ['webtty']);

    this.ws.onopen = () => {
      console.log('WebSocket connected');

      // Send auth token
      const authToken = window.gotty_auth_token || '';
      this.ws.send(JSON.stringify({ AuthToken: authToken, Arguments: '' }));

      // Tell server to expect base64 encoded input
      this.sendMessage(MSG.SetEncoding, 'base64');

      // Send initial size and focus terminal
      setTimeout(() => {
        this.sendResize();
        this.terminal.focus();
      }, 100);

      // Switch to pending session if we reconnected after session ended
      if (this.pendingSessionSwitch) {
        setTimeout(() => {
          console.log('Switching to session:', this.pendingSessionSwitch);
          this.switchSession(this.pendingSessionSwitch);
          this.pendingSessionSwitch = null;
        }, 200);
      }
    };

    this.ws.onmessage = (event) => {
      this.handleMessage(event.data);
    };

    this.ws.onclose = () => {
      console.log('WebSocket closed');

      // Check if there are other sessions to switch to
      const otherSessions = this.layout?.sessions?.filter(s => !s.active) || [];
      if (otherSessions.length > 0) {
        // Auto-reconnect and switch to another session
        this.pendingSessionSwitch = otherSessions[0].name;
        console.log('Auto-reconnecting to session:', this.pendingSessionSwitch);
        setTimeout(() => this.connect(), 500);
      } else if (this.reconnectInterval) {
        // Normal reconnect behavior
        setTimeout(() => this.connect(), this.reconnectInterval * 1000);
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  handleMessage(data) {
    const type = data[0];
    const payload = data.slice(1);

    switch (type) {
      case MSG.Output:
        // Decode base64 to Uint8Array for proper UTF-8 handling
        const binaryString = atob(payload);

        // Check for OSC 52 clipboard sequences and handle them
        const processed = this.handleOSC52(binaryString);

        const bytes = new Uint8Array(processed.length);
        for (let i = 0; i < processed.length; i++) {
          bytes[i] = processed.charCodeAt(i);
        }
        this.terminal.write(bytes);
        break;

      case MSG.Pong:
        // Ignore pong
        break;

      case MSG.SetWindowTitle:
        document.title = payload;
        break;

      case MSG.SetPreferences:
        const prefs = JSON.parse(payload);
        if (prefs.fontSize) {
          this.terminal.options.fontSize = prefs.fontSize;
          this.fitAddon.fit();
        }
        break;

      case MSG.SetReconnect:
        this.reconnectInterval = parseInt(payload, 10);
        break;

      case MSG.SetBufferSize:
        this.bufferSize = parseInt(payload, 10);
        break;

      case MSG.TmuxLayoutUpdate:
        this.layout = JSON.parse(payload);
        this.dispatchLayoutUpdate();
        break;

      case MSG.TmuxModeUpdate:
        const modeState = JSON.parse(payload);
        this.inCopyMode = modeState.inCopyMode;
        break;

      default:
        console.warn('Unknown message type:', type);
    }
  }

  sendMessage(type, payload = '') {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(type + payload);
    } else {
      console.warn('WebSocket not ready, state:', this.ws?.readyState);
    }
  }

  sendResize() {
    const dims = { columns: this.terminal.cols, rows: this.terminal.rows };
    this.sendMessage(MSG.ResizeTerminal, JSON.stringify(dims));
  }

  dispatchLayoutUpdate() {
    // Notify sidebar and other components
    window.dispatchEvent(new CustomEvent('tmux-layout-update', {
      detail: this.layout
    }));
  }

  // Public API for components
  selectPane(paneId) {
    this.sendMessage(MSG.TmuxSelectPane, paneId);
  }

  selectWindow(windowId) {
    this.sendMessage(MSG.TmuxSelectWindow, windowId);
  }

  splitPane(horizontal) {
    this.sendMessage(MSG.TmuxSplitPane, horizontal ? 'h' : 'v');
  }

  closePane(paneId) {
    this.sendMessage(MSG.TmuxClosePane, paneId);
  }

  newWindow() {
    this.sendMessage(MSG.TmuxNewWindow, '');
  }

  switchSession(sessionName) {
    this.sendMessage(MSG.TmuxSwitchSession, sessionName);
  }

  enterCopyMode() {
    this.sendMessage(MSG.TmuxCopyMode, '1');
    this.inCopyMode = true;
  }

  exitCopyMode() {
    this.sendMessage(MSG.TmuxCopyMode, '0');
    this.inCopyMode = false;
  }

  // Send raw key bytes to terminal (used by shortcuts component)
  sendKeys(keys) {
    const bytes = new Uint8Array(keys);
    const binary = String.fromCharCode(...bytes);
    this.sendMessage(MSG.Input, btoa(binary));
  }

  // Handle OSC 52 clipboard sequences from tmux
  // Format: ESC ] 52 ; Pc ; Pd BEL  or  ESC ] 52 ; Pc ; Pd ESC \
  handleOSC52(data) {
    const ESC = String.fromCharCode(0x1b);
    const oscStart = ESC + ']52;';
    let result = data;
    let startIdx = data.indexOf(oscStart);

    while (startIdx !== -1) {
      // Find the terminator (BEL \x07 or ST \x1b\\)
      let endIdx = -1;
      let termLen = 1;

      for (let i = startIdx + oscStart.length; i < data.length; i++) {
        if (data.charCodeAt(i) === 0x07) { // BEL
          endIdx = i;
          termLen = 1;
          break;
        }
        if (data.charCodeAt(i) === 0x1b && i + 1 < data.length && data[i + 1] === '\\') { // ST
          endIdx = i;
          termLen = 2;
          break;
        }
      }

      if (endIdx === -1) break;

      // Extract the content between start and terminator
      const content = data.substring(startIdx + oscStart.length, endIdx);

      // Content format: Pc;Pd where Pc is selection and Pd is base64 data
      const semiIdx = content.indexOf(';');
      if (semiIdx !== -1) {
        const base64Data = content.substring(semiIdx + 1);

        if (base64Data && base64Data !== '?') {
          try {
            // Decode base64 to bytes, then UTF-8 decode for proper emoji support
            const binaryStr = atob(base64Data);
            const bytes = Uint8Array.from(binaryStr, c => c.charCodeAt(0));
            const text = new TextDecoder('utf-8').decode(bytes);
            navigator.clipboard.writeText(text);
          } catch (e) {
            // Silently ignore decode errors
          }
        }
      }

      // Remove this OSC sequence from output
      const fullSeq = data.substring(startIdx, endIdx + termLen);
      result = result.replace(fullSeq, '');

      // Look for more
      startIdx = data.indexOf(oscStart, startIdx + 1);
    }

    return result;
  }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  new WebTmux();
});
