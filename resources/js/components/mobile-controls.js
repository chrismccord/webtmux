// Mobile controls component
import { LitElement, html, css } from 'lit';

class WebtmuxMobileControls extends LitElement {
  static properties = {
    showPaneSelector: { type: Boolean },
    showSessionSelector: { type: Boolean },
    layout: { type: Object },
  };

  static styles = css`
    :host {
      display: block;
      position: fixed;
      bottom: 0;
      left: 0;
      right: 0;
      background: #16213e;
      border-top: 1px solid #0f3460;
      padding: 6px;
      padding-bottom: calc(6px + env(safe-area-inset-bottom));
      z-index: 1000;
    }

    .controls {
      display: flex;
      justify-content: space-around;
      gap: 8px;
    }

    .control-btn {
      flex: 1;
      max-width: 60px;
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 6px;
      color: #888;
      padding: 8px 4px;
      font-size: 9px;
      cursor: pointer;
      transition: all 0.2s;
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 2px;
      -webkit-tap-highlight-color: transparent;
    }

    .control-btn:active {
      background: #0f3460;
      border-color: #e94560;
      color: #fff;
      transform: scale(0.95);
    }

    .control-btn svg {
      width: 16px;
      height: 16px;
    }

    .control-btn.prefix {
      background: #e94560;
      border-color: #e94560;
      color: #fff;
    }

    .window-tabs {
      display: flex;
      gap: 4px;
      margin-bottom: 8px;
      overflow-x: auto;
      padding-bottom: 4px;
      -webkit-overflow-scrolling: touch;
    }

    .window-tab {
      flex-shrink: 0;
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 4px;
      color: #888;
      padding: 6px 12px;
      font-size: 11px;
      cursor: pointer;
      white-space: nowrap;
    }

    .window-tab:active, .window-tab.active {
      background: #e94560;
      border-color: #e94560;
      color: #fff;
    }

    .pane-selector {
      position: absolute;
      bottom: 100%;
      left: 0;
      right: 0;
      background: #16213e;
      border-top: 1px solid #0f3460;
      padding: 12px;
      display: none;
    }

    .pane-selector.open {
      display: block;
    }

    .pane-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 8px;
    }

    .pane-btn {
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 4px;
      color: #888;
      padding: 12px;
      font-size: 12px;
      cursor: pointer;
    }

    .pane-btn.active {
      border-color: #e94560;
      color: #e94560;
    }

    .arrow-pad {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      grid-template-rows: repeat(3, 1fr);
      gap: 2px;
      width: 90px;
      height: 90px;
    }

    .arrow-pad-wrap {
      display: flex;
      justify-content: center;
      margin-top: 8px;
    }

    .arrow-btn {
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 4px;
      color: #888;
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      font-size: 14px;
    }

    .arrow-btn:active {
      background: #0f3460;
      border-color: #e94560;
    }

    .arrow-btn.empty {
      visibility: hidden;
    }

    .session-overlay {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(0, 0, 0, 0.8);
      display: none;
      align-items: center;
      justify-content: center;
      z-index: 2000;
    }

    .session-overlay.open {
      display: flex;
    }

    .session-modal {
      background: #16213e;
      border: 1px solid #0f3460;
      border-radius: 12px;
      padding: 20px;
      min-width: 280px;
      max-width: 90%;
      max-height: 70vh;
      overflow-y: auto;
    }

    .session-modal h3 {
      color: #4a9eff;
      font-size: 14px;
      margin: 0 0 16px 0;
      text-align: center;
    }

    .session-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    .session-item {
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 8px;
      padding: 12px 16px;
      color: #888;
      font-size: 14px;
      cursor: pointer;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .session-item:active {
      background: #0f3460;
      border-color: #4a9eff;
    }

    .session-item.active {
      border-color: #4a9eff;
      color: #fff;
      background: #1a3a5c;
    }

    .session-item .session-name {
      font-weight: 500;
    }

    .session-item .session-meta {
      font-size: 12px;
      opacity: 0.7;
    }

    .close-overlay {
      position: absolute;
      top: 20px;
      right: 20px;
      background: transparent;
      border: none;
      color: #888;
      font-size: 24px;
      cursor: pointer;
    }

    .session-btn {
      background: #4a9eff;
      border-color: #4a9eff;
      color: #fff;
    }
  `;

  constructor() {
    super();
    this.showPaneSelector = false;
    this.showSessionSelector = false;
    this.layout = null;

    window.addEventListener('tmux-layout-update', (e) => {
      this.layout = e.detail;
    });
  }

  render() {
    const sessions = this.layout?.sessions || [];
    const showSessionBtn = sessions.length > 1;

    return html`
      <!-- Session overlay -->
      <div class="session-overlay ${this.showSessionSelector ? 'open' : ''}" @click=${this.closeSessionSelector}>
        <div class="session-modal" @click=${(e) => e.stopPropagation()}>
          <h3>Switch Session</h3>
          <div class="session-list">
            ${sessions.map(sess => html`
              <button
                class="session-item ${sess.active ? 'active' : ''}"
                @click=${() => this.switchSession(sess.name)}
              >
                <span class="session-name">${sess.name}</span>
                <span class="session-meta">${sess.windows} window${sess.windows !== 1 ? 's' : ''}</span>
              </button>
            `)}
          </div>
        </div>
      </div>

      <div class="pane-selector ${this.showPaneSelector ? 'open' : ''}">
        <div class="pane-grid">
          ${this.layout?.windows?.find(w => w.active)?.panes?.map(pane => html`
            <button
              class="pane-btn ${pane.active ? 'active' : ''}"
              @click=${() => this.selectPane(pane.id)}
            >
              Pane ${pane.index}
            </button>
          `)}
        </div>
      </div>

      ${this.layout?.windows?.length > 0 ? html`
        <div class="window-tabs">
          ${this.layout.windows.map(win => html`
            <button
              class="window-tab ${win.active ? 'active' : ''}"
              @click=${() => this.selectWindow(win.id)}
            >
              ${win.index}: ${win.name || 'bash'}
            </button>
          `)}
          <button class="window-tab" @click=${this.newWindow}>+</button>
        </div>
      ` : ''}

      <div class="controls">
        ${showSessionBtn ? html`
          <button class="control-btn session-btn" @click=${this.toggleSessionSelector}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="3" width="20" height="14" rx="2"/>
              <line x1="8" y1="21" x2="16" y2="21"/>
              <line x1="12" y1="17" x2="12" y2="21"/>
            </svg>
            Sess
          </button>
        ` : ''}

        <button class="control-btn prefix" @click=${this.sendPrefix}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <text x="12" y="16" font-size="10" fill="currentColor" text-anchor="middle">^B</text>
          </svg>
          Prefix
        </button>

        <button class="control-btn" @click=${() => this.splitPane(true)}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <line x1="12" y1="3" x2="12" y2="21"/>
          </svg>
          Split H
        </button>

        <button class="control-btn" @click=${() => this.splitPane(false)}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <line x1="3" y1="12" x2="21" y2="12"/>
          </svg>
          Split V
        </button>

        <button class="control-btn" @click=${this.togglePaneSelector}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7"/>
            <rect x="14" y="3" width="7" height="7"/>
            <rect x="3" y="14" width="7" height="7"/>
            <rect x="14" y="14" width="7" height="7"/>
          </svg>
          Panes
        </button>

        <button class="control-btn" @click=${this.newWindow}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <line x1="12" y1="8" x2="12" y2="16"/>
            <line x1="8" y1="12" x2="16" y2="12"/>
          </svg>
          New
        </button>
      </div>

      <div class="arrow-pad-wrap">
        <div class="arrow-pad">
          <button class="arrow-btn empty"></button>
          <button class="arrow-btn" @click=${() => this.sendArrow('up')}>▲</button>
          <button class="arrow-btn empty"></button>
          <button class="arrow-btn" @click=${() => this.sendArrow('left')}>◀</button>
          <button class="arrow-btn empty"></button>
          <button class="arrow-btn" @click=${() => this.sendArrow('right')}>▶</button>
          <button class="arrow-btn empty"></button>
          <button class="arrow-btn" @click=${() => this.sendArrow('down')}>▼</button>
          <button class="arrow-btn empty"></button>
        </div>
      </div>
    `;
  }

  sendPrefix() {
    // Send Ctrl+B (tmux prefix)
    // ASCII code for Ctrl+B is 0x02
    window.webtmux?.terminal?.input('\x02');
  }

  sendArrow(direction) {
    const arrows = {
      up: '\x1b[A',
      down: '\x1b[B',
      right: '\x1b[C',
      left: '\x1b[D',
    };
    const sequence = arrows[direction];
    if (!sequence) {
      return;
    }
    window.webtmux?.sendInput?.(sequence);
  }

  splitPane(horizontal) {
    window.webtmux?.splitPane(horizontal);
  }

  togglePaneSelector() {
    this.showPaneSelector = !this.showPaneSelector;
  }

  selectPane(paneId) {
    window.webtmux?.selectPane(paneId);
    this.showPaneSelector = false;
  }

  selectWindow(windowId) {
    window.webtmux?.selectWindow(windowId);
  }

  newWindow() {
    window.webtmux?.newWindow();
  }

  toggleSessionSelector() {
    this.showSessionSelector = !this.showSessionSelector;
  }

  closeSessionSelector() {
    this.showSessionSelector = false;
  }

  switchSession(sessionName) {
    window.webtmux?.switchSession(sessionName);
    this.showSessionSelector = false;
  }
}

customElements.define('webtmux-mobile-controls', WebtmuxMobileControls);
