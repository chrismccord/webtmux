// Shortcuts bar component with settings modal
import { LitElement, html, css } from 'lit';
import {
  getShortcuts,
  resetToDefaults,
  addShortcut,
  removeShortcut,
  moveShortcut,
  toggleShortcut,
  updateShortcut,
  KEY_TEMPLATES,
} from '../services/shortcuts-service.js';

class WebtmuxShortcuts extends LitElement {
  static properties = {
    shortcuts: { type: Array },
    showSettings: { type: Boolean },
    showAddForm: { type: Boolean },
    editingId: { type: String },
    newLabel: { type: String },
    newKeys: { type: Array },
    selectedTemplate: { type: String },
  };

  static styles = css`
    :host {
      display: block;
      background: #16213e;
      border-top: 1px solid #0f3460;
      padding: 6px 8px;
    }

    .shortcuts-bar {
      display: flex;
      gap: 6px;
      overflow-x: auto;
      align-items: center;
      -webkit-overflow-scrolling: touch;
      scrollbar-width: none;
    }

    .shortcuts-bar::-webkit-scrollbar {
      display: none;
    }

    .shortcut-btn {
      flex-shrink: 0;
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 4px;
      color: #888;
      padding: 6px 10px;
      font-size: 12px;
      font-family: monospace;
      cursor: pointer;
      transition: all 0.15s;
      min-width: 36px;
      text-align: center;
      -webkit-tap-highlight-color: transparent;
    }

    /* Tablet: iPad Mini landscape (1024px) and tablets up to 1279px */
    @media (min-width: 768px) and (max-width: 1279px) {
      :host {
        padding: 10px 16px;
      }

      .shortcuts-bar {
        gap: 12px;
      }

      .shortcut-btn {
        min-width: 56px;
        min-height: 50px;
        padding: 12px 18px;
        font-size: 16px;
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
      }
    }

    /* Phone: Smaller screens (< 768px) */
    @media (max-width: 767px) {
      :host {
        padding: 8px 10px;
      }

      .shortcuts-bar {
        gap: 8px;
      }

      .shortcut-btn {
        min-width: 44px;
        min-height: 44px;
        padding: 10px 14px;
        font-size: 14px;
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
      }
    }

    .shortcut-btn:hover {
      border-color: #e94560;
      color: #fff;
    }

    .shortcut-btn:active {
      background: #0f3460;
      transform: scale(0.95);
    }

    .settings-btn {
      flex-shrink: 0;
      background: transparent;
      border: 1px solid #0f3460;
      border-radius: 4px;
      color: #666;
      padding: 6px 8px;
      font-size: 14px;
      cursor: pointer;
      transition: all 0.15s;
      margin-left: auto;
    }

    .settings-btn:hover {
      border-color: #4a9eff;
      color: #4a9eff;
    }

    /* Tablet: Larger settings button */
    @media (min-width: 768px) and (max-width: 1023px) {
      .settings-btn {
        min-width: 50px;
        min-height: 50px;
        padding: 12px;
        font-size: 20px;
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
      }
    }

    /* Phone: Settings button */
    @media (max-width: 767px) {
      .settings-btn {
        min-width: 44px;
        min-height: 44px;
        padding: 10px;
        font-size: 18px;
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
      }
    }

    /* Settings Modal */
    .modal-overlay {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(0, 0, 0, 0.85);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 2000;
    }

    .modal {
      background: #16213e;
      border: 1px solid #0f3460;
      border-radius: 12px;
      padding: 20px;
      width: 90%;
      max-width: 420px;
      max-height: 80vh;
      overflow-y: auto;
    }

    .modal h2 {
      color: #e94560;
      font-size: 16px;
      margin: 0 0 16px 0;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }

    .close-btn {
      background: transparent;
      border: none;
      color: #888;
      font-size: 20px;
      cursor: pointer;
      padding: 0;
      line-height: 1;
    }

    .close-btn:hover {
      color: #fff;
    }

    .shortcut-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
      margin-bottom: 16px;
    }

    .shortcut-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px;
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 6px;
    }

    .shortcut-item.disabled {
      opacity: 0.5;
    }

    .shortcut-toggle {
      width: 18px;
      height: 18px;
      cursor: pointer;
      accent-color: #e94560;
    }

    .shortcut-label {
      flex: 1;
      color: #ccc;
      font-size: 13px;
      font-family: monospace;
    }

    .shortcut-keys {
      color: #666;
      font-size: 10px;
      font-family: monospace;
    }

    .shortcut-actions {
      display: flex;
      gap: 4px;
    }

    .action-btn {
      background: #0f3460;
      border: none;
      border-radius: 4px;
      color: #888;
      padding: 4px 6px;
      font-size: 11px;
      cursor: pointer;
      transition: all 0.15s;
    }

    .action-btn:hover {
      background: #1a4a80;
      color: #fff;
    }

    .action-btn.delete:hover {
      background: #e94560;
    }

    .modal-actions {
      display: flex;
      gap: 8px;
      margin-top: 16px;
      padding-top: 16px;
      border-top: 1px solid #0f3460;
    }

    .modal-btn {
      flex: 1;
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 6px;
      color: #888;
      padding: 10px;
      font-size: 12px;
      cursor: pointer;
      transition: all 0.15s;
    }

    .modal-btn:hover {
      border-color: #4a9eff;
      color: #fff;
    }

    .modal-btn.primary {
      background: #4a9eff;
      border-color: #4a9eff;
      color: #fff;
    }

    .modal-btn.primary:hover {
      background: #3a8eef;
    }

    .modal-btn.danger {
      border-color: #e94560;
      color: #e94560;
    }

    .modal-btn.danger:hover {
      background: #e94560;
      color: #fff;
    }

    /* Add Form */
    .add-form {
      background: #1a1a2e;
      border: 1px solid #0f3460;
      border-radius: 8px;
      padding: 12px;
      margin-bottom: 16px;
    }

    .add-form h3 {
      color: #4a9eff;
      font-size: 13px;
      margin: 0 0 12px 0;
    }

    .form-row {
      margin-bottom: 12px;
    }

    .form-row label {
      display: block;
      color: #888;
      font-size: 11px;
      margin-bottom: 4px;
    }

    .form-row input,
    .form-row select {
      width: 100%;
      background: #0f3460;
      border: 1px solid #1a4a80;
      border-radius: 4px;
      color: #fff;
      padding: 8px;
      font-size: 13px;
      box-sizing: border-box;
    }

    .form-row input:focus,
    .form-row select:focus {
      outline: none;
      border-color: #4a9eff;
    }

    .form-row select {
      cursor: pointer;
    }

    .custom-keys-input {
      font-family: monospace;
      font-size: 12px;
    }

    .form-hint {
      color: #666;
      font-size: 10px;
      margin-top: 4px;
    }

    .form-actions {
      display: flex;
      gap: 8px;
      margin-top: 12px;
    }

    .form-actions button {
      flex: 1;
      padding: 8px;
      border-radius: 4px;
      font-size: 12px;
      cursor: pointer;
      border: 1px solid #0f3460;
      transition: all 0.15s;
    }

    .form-actions .cancel-btn {
      background: #1a1a2e;
      color: #888;
    }

    .form-actions .cancel-btn:hover {
      border-color: #888;
      color: #fff;
    }

    .form-actions .save-btn {
      background: #4a9eff;
      border-color: #4a9eff;
      color: #fff;
    }

    .form-actions .save-btn:hover {
      background: #3a8eef;
    }

    .form-actions .save-btn:disabled {
      background: #333;
      border-color: #333;
      color: #666;
      cursor: not-allowed;
    }
  `;

  constructor() {
    super();
    this.shortcuts = getShortcuts();
    this.showSettings = false;
    this.showAddForm = false;
    this.editingId = null;
    this.newLabel = '';
    this.newKeys = [];
    this.selectedTemplate = '';
  }

  render() {
    const enabledShortcuts = this.shortcuts
      .filter(s => s.enabled)
      .sort((a, b) => a.order - b.order);

    return html`
      <div class="shortcuts-bar">
        ${enabledShortcuts.map(shortcut => html`
          <button
            class="shortcut-btn"
            @click=${() => this.sendShortcut(shortcut)}
            title="${this.formatKeys(shortcut.keys)}"
          >
            ${shortcut.label}
          </button>
        `)}
        <button class="settings-btn" @click=${this.toggleSettings} title="Configure shortcuts">
          \u2699
        </button>
      </div>

      ${this.showSettings ? this.renderSettingsModal() : ''}
    `;
  }

  renderSettingsModal() {
    const sortedShortcuts = [...this.shortcuts].sort((a, b) => a.order - b.order);

    return html`
      <div class="modal-overlay" @click=${this.handleOverlayClick}>
        <div class="modal" @click=${e => e.stopPropagation()}>
          <h2>
            Keyboard Shortcuts
            <button class="close-btn" @click=${this.toggleSettings}>\u00d7</button>
          </h2>

          ${this.showAddForm ? this.renderAddForm() : ''}

          <div class="shortcut-list">
            ${sortedShortcuts.map((shortcut, index) => html`
              <div class="shortcut-item ${shortcut.enabled ? '' : 'disabled'}">
                <input
                  type="checkbox"
                  class="shortcut-toggle"
                  .checked=${shortcut.enabled}
                  @change=${() => this.handleToggle(shortcut.id)}
                />
                <span class="shortcut-label">${shortcut.label}</span>
                <span class="shortcut-keys">${this.formatKeys(shortcut.keys)}</span>
                <div class="shortcut-actions">
                  <button
                    class="action-btn"
                    @click=${() => this.handleMove(shortcut.id, 'up')}
                    ?disabled=${index === 0}
                    title="Move up"
                  >\u2191</button>
                  <button
                    class="action-btn"
                    @click=${() => this.handleMove(shortcut.id, 'down')}
                    ?disabled=${index === sortedShortcuts.length - 1}
                    title="Move down"
                  >\u2193</button>
                  ${!shortcut.builtin ? html`
                    <button
                      class="action-btn delete"
                      @click=${() => this.handleDelete(shortcut.id)}
                      title="Delete"
                    >\u00d7</button>
                  ` : ''}
                </div>
              </div>
            `)}
          </div>

          <div class="modal-actions">
            <button class="modal-btn primary" @click=${this.handleAddClick}>
              + Add Shortcut
            </button>
            <button class="modal-btn danger" @click=${this.handleReset}>
              Reset
            </button>
          </div>
        </div>
      </div>
    `;
  }

  renderAddForm() {
    const isCustom = this.selectedTemplate === 'custom';

    return html`
      <div class="add-form">
        <h3>${this.editingId ? 'Edit Shortcut' : 'Add New Shortcut'}</h3>

        <div class="form-row">
          <label>Label</label>
          <input
            type="text"
            .value=${this.newLabel}
            @input=${e => this.newLabel = e.target.value}
            placeholder="e.g., Enter, ^Z"
            maxlength="10"
          />
        </div>

        <div class="form-row">
          <label>Key Sequence</label>
          <select
            .value=${this.selectedTemplate}
            @change=${this.handleTemplateChange}
          >
            <option value="">-- Select a key --</option>
            ${KEY_TEMPLATES.map(t => html`
              <option value=${t.label}>${t.label}</option>
            `)}
            <option value="custom">Custom (hex)</option>
          </select>
        </div>

        ${isCustom ? html`
          <div class="form-row">
            <label>Custom Key Bytes (hex, comma-separated)</label>
            <input
              type="text"
              class="custom-keys-input"
              .value=${this.newKeys.map(k => '0x' + k.toString(16).padStart(2, '0')).join(', ')}
              @input=${this.handleCustomKeysInput}
              placeholder="e.g., 0x1b, 0x5b, 0x41"
            />
            <div class="form-hint">Example: 0x1b for ESC, 0x0d for Enter</div>
          </div>
        ` : ''}

        <div class="form-actions">
          <button class="cancel-btn" @click=${this.handleCancelAdd}>Cancel</button>
          <button
            class="save-btn"
            @click=${this.handleSaveAdd}
            ?disabled=${!this.newLabel.trim() || this.newKeys.length === 0}
          >
            ${this.editingId ? 'Update' : 'Add'}
          </button>
        </div>
      </div>
    `;
  }

  formatKeys(keys) {
    return keys.map(k => '0x' + k.toString(16).toUpperCase().padStart(2, '0')).join(' ');
  }

  sendShortcut(shortcut) {
    const bytes = new Uint8Array(shortcut.keys);
    const binary = String.fromCharCode(...bytes);
    if (window.webtmux?.sendKeys) {
      window.webtmux.sendKeys(shortcut.keys);
    } else {
      // Fallback: send directly
      window.webtmux?.sendMessage?.('1', btoa(binary));
    }
    // Re-focus terminal after clicking button
    window.webtmux?.terminal?.focus();
  }

  toggleSettings() {
    this.showSettings = !this.showSettings;
    if (!this.showSettings) {
      this.showAddForm = false;
      this.editingId = null;
    }
  }

  handleOverlayClick(e) {
    if (e.target === e.currentTarget) {
      this.toggleSettings();
    }
  }

  handleToggle(id) {
    this.shortcuts = toggleShortcut(this.shortcuts, id);
  }

  handleMove(id, direction) {
    this.shortcuts = moveShortcut(this.shortcuts, id, direction);
  }

  handleDelete(id) {
    this.shortcuts = removeShortcut(this.shortcuts, id);
  }

  handleAddClick() {
    this.showAddForm = true;
    this.editingId = null;
    this.newLabel = '';
    this.newKeys = [];
    this.selectedTemplate = '';
  }

  handleCancelAdd() {
    this.showAddForm = false;
    this.editingId = null;
    this.newLabel = '';
    this.newKeys = [];
    this.selectedTemplate = '';
  }

  handleTemplateChange(e) {
    this.selectedTemplate = e.target.value;
    if (this.selectedTemplate && this.selectedTemplate !== 'custom') {
      const template = KEY_TEMPLATES.find(t => t.label === this.selectedTemplate);
      if (template) {
        this.newKeys = [...template.keys];
        if (!this.newLabel.trim()) {
          this.newLabel = template.label;
        }
      }
    } else if (this.selectedTemplate === 'custom') {
      this.newKeys = [];
    }
  }

  handleCustomKeysInput(e) {
    const input = e.target.value;
    // Parse comma-separated hex values
    const parts = input.split(',').map(s => s.trim()).filter(s => s);
    const keys = [];
    for (const part of parts) {
      const num = parseInt(part, 16);
      if (!isNaN(num) && num >= 0 && num <= 255) {
        keys.push(num);
      }
    }
    this.newKeys = keys;
  }

  handleSaveAdd() {
    if (!this.newLabel.trim() || this.newKeys.length === 0) return;

    if (this.editingId) {
      this.shortcuts = updateShortcut(this.shortcuts, this.editingId, {
        label: this.newLabel.trim(),
        keys: [...this.newKeys],
      });
    } else {
      this.shortcuts = addShortcut(this.shortcuts, this.newLabel.trim(), [...this.newKeys]);
    }

    this.handleCancelAdd();
  }

  handleReset() {
    if (confirm('Reset all shortcuts to defaults? Custom shortcuts will be removed.')) {
      this.shortcuts = resetToDefaults();
      this.showAddForm = false;
    }
  }
}

customElements.define('webtmux-shortcuts', WebtmuxShortcuts);
