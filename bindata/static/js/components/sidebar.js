// Sidebar component with minimap
import { LitElement, html, css } from 'lit';

class WebtmuxSidebar extends LitElement {
  static properties = {
    layout: { type: Object },
    activePane: { type: String },
    activeWindow: { type: String },
    collapsed: { type: Boolean },
  };

  static styles = css`
    :host {
      display: block;
      width: 220px;
      background: #16213e;
      border-left: 1px solid #0f3460;
      padding: 12px;
      overflow-y: auto;
      transition: width 0.2s, padding 0.2s;
    }

    :host(.collapsed) {
      width: 40px;
      padding: 8px;
      overflow: hidden;
    }

    .toggle-btn {
      position: absolute;
      top: 8px;
      right: 8px;
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 4px;
      color: #888;
      width: 24px;
      height: 24px;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 10;
    }

    .toggle-btn:hover {
      border-color: #e94560;
      color: #fff;
    }

    :host(.collapsed) .toggle-btn {
      position: static;
      margin: 0 auto;
    }

    :host(.collapsed) .sidebar-content {
      display: none;
    }

    .sidebar-content {
      position: relative;
    }

    h3 {
      color: #e94560;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 1px;
      margin: 0 0 12px 0;
    }

    .window-tabs {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      margin-bottom: 16px;
    }

    .window-tab {
      background: #1a1a2e;
      color: #888;
      border: 1px solid #0f3460;
      border-radius: 4px;
      padding: 4px 8px;
      font-size: 11px;
      cursor: pointer;
      transition: all 0.2s;
    }

    .window-tab:hover {
      border-color: #e94560;
      color: #fff;
    }

    .window-tab.active {
      background: #e94560;
      border-color: #e94560;
      color: #fff;
    }

    .minimap {
      position: relative;
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 4px;
      height: 150px;
      margin-bottom: 16px;
    }

    .pane {
      position: absolute;
      background: #0f3460;
      border: 1px solid #16213e;
      border-radius: 2px;
      cursor: pointer;
      transition: all 0.2s;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 10px;
      color: #666;
    }

    .pane:hover {
      border-color: #e94560;
      background: #1a3a5c;
    }

    .pane.active {
      border-color: #e94560;
      background: #1a3a5c;
      box-shadow: 0 0 8px rgba(233, 69, 96, 0.3);
    }

    .pane.active::after {
      content: '';
      position: absolute;
      top: 2px;
      right: 2px;
      width: 6px;
      height: 6px;
      background: #e94560;
      border-radius: 50%;
    }

    .actions {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 8px;
    }

    .action-btn {
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 4px;
      color: #888;
      padding: 8px;
      font-size: 11px;
      cursor: pointer;
      transition: all 0.2s;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 4px;
    }

    .action-btn:hover {
      border-color: #e94560;
      color: #fff;
    }

    .action-btn svg {
      width: 14px;
      height: 14px;
    }

    .session-info {
      color: #666;
      font-size: 10px;
      margin-top: 16px;
      padding-top: 12px;
      border-top: 1px solid #0f3460;
    }

    .session-tabs {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      margin-bottom: 16px;
    }

    .session-tab {
      background: #1a1a2e;
      color: #888;
      border: 1px solid #0f3460;
      border-radius: 4px;
      padding: 4px 8px;
      font-size: 11px;
      cursor: pointer;
      transition: all 0.2s;
    }

    .session-tab:hover {
      border-color: #4a9eff;
      color: #fff;
    }

    .session-tab.active {
      background: #4a9eff;
      border-color: #4a9eff;
      color: #fff;
    }

    .session-tab .win-count {
      font-size: 9px;
      opacity: 0.7;
      margin-left: 4px;
    }
  `;

  constructor() {
    super();
    this.layout = null;
    this.activePane = '';
    this.activeWindow = '';
    this.collapsed = false;

    // Listen for layout updates
    window.addEventListener('tmux-layout-update', (e) => {
      this.layout = e.detail;
      this.activePane = e.detail.activePaneId;
      this.activeWindow = e.detail.activeWindowId;
    });
  }

  updated(changedProperties) {
    if (changedProperties.has('collapsed')) {
      if (this.collapsed) {
        this.classList.add('collapsed');
      } else {
        this.classList.remove('collapsed');
      }
    }
  }

  toggleCollapsed() {
    this.collapsed = !this.collapsed;
  }

  render() {
    const toggleIcon = this.collapsed
      ? html`<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>`
      : html`<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>`;

    if (!this.layout) {
      return html`
        <button class="toggle-btn" @click=${this.toggleCollapsed}>${toggleIcon}</button>
        <div class="sidebar-content">
          <h3>tmux</h3>
          <p style="color: #666; font-size: 12px;">Connecting...</p>
        </div>
      `;
    }

    const activeWindow = this.layout.windows?.find(w => w.id === this.activeWindow);

    const sessions = this.layout.sessions || [];
    const showSessions = sessions.length > 1;

    return html`
      <button class="toggle-btn" @click=${this.toggleCollapsed}>${toggleIcon}</button>
      <div class="sidebar-content">
      ${showSessions ? html`
        <h3>Sessions</h3>
        <div class="session-tabs">
          ${sessions.map(sess => html`
            <button
              class="session-tab ${sess.active ? 'active' : ''}"
              @click=${() => this.switchSession(sess.name)}
            >
              ${sess.name}<span class="win-count">(${sess.windows})</span>
            </button>
          `)}
        </div>
      ` : ''}

      <h3>Windows</h3>
      <div class="window-tabs">
        ${this.layout.windows?.map(win => html`
          <button
            class="window-tab ${win.id === this.activeWindow ? 'active' : ''}"
            @click=${() => this.selectWindow(win.id)}
          >
            ${win.index}: ${win.name || 'bash'}
          </button>
        `)}
        <button class="window-tab" @click=${() => this.newWindow()}>+</button>
      </div>

      <h3>Panes</h3>
      <div class="minimap">
        ${activeWindow?.panes?.map(pane => {
          // Calculate percentage positions
          const totalWidth = activeWindow.panes.reduce((max, p) => Math.max(max, p.left + p.width), 0);
          const totalHeight = activeWindow.panes.reduce((max, p) => Math.max(max, p.top + p.height), 0);

          const left = (pane.left / totalWidth) * 100;
          const top = (pane.top / totalHeight) * 100;
          const width = (pane.width / totalWidth) * 100;
          const height = (pane.height / totalHeight) * 100;

          return html`
            <div
              class="pane ${pane.id === this.activePane ? 'active' : ''}"
              style="left: ${left}%; top: ${top}%; width: ${width}%; height: ${height}%"
              @click=${() => this.selectPane(pane.id)}
              title="${pane.command}"
            >
              ${pane.index}
            </div>
          `;
        })}
      </div>

      <h3>Actions</h3>
      <div class="actions">
        <button class="action-btn" @click=${() => this.splitPane(true)}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <line x1="12" y1="3" x2="12" y2="21"/>
          </svg>
          Split H
        </button>
        <button class="action-btn" @click=${() => this.splitPane(false)}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <line x1="3" y1="12" x2="21" y2="12"/>
          </svg>
          Split V
        </button>
        <button class="action-btn" @click=${() => this.newWindow()}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <line x1="12" y1="8" x2="12" y2="16"/>
            <line x1="8" y1="12" x2="16" y2="12"/>
          </svg>
          New Win
        </button>
        <button class="action-btn" @click=${() => this.closePane()}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
          Close
        </button>
      </div>

      <div class="session-info">
        Session: ${this.layout.sessionName}<br>
        ${this.layout.windows?.length || 0} windows, ${activeWindow?.panes?.length || 0} panes
      </div>
      </div>
    `;
  }

  selectPane(paneId) {
    window.webtmux?.selectPane(paneId);
  }

  selectWindow(windowId) {
    window.webtmux?.selectWindow(windowId);
  }

  switchSession(sessionName) {
    window.webtmux?.switchSession(sessionName);
  }

  splitPane(horizontal) {
    window.webtmux?.splitPane(horizontal);
  }

  newWindow() {
    window.webtmux?.newWindow();
  }

  closePane() {
    window.webtmux?.closePane(this.activePane);
  }
}

customElements.define('webtmux-sidebar', WebtmuxSidebar);
