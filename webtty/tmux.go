package webtty

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
	"webtmux/pkg/tmux"
)

// TmuxController interface for tmux operations
type TmuxController interface {
	GetLayout() *tmux.Layout
	RefreshLayout() error
	SelectPane(paneID string) error
	SelectWindow(windowID string) error
	SwitchSession(sessionName string) error
	SplitPane(horizontal bool) error
	ClosePane(paneID string) error
	EnterCopyMode() error
	ExitCopyMode() error
	ScrollUp(lines int) error
	ScrollDown(lines int) error
	NewWindow() error
	Events() <-chan tmux.Event
}

// SetTmuxController sets the tmux controller for the WebTTY instance
func (wt *WebTTY) SetTmuxController(tc TmuxController) {
	wt.tmuxCtrl = tc
}

// SendTmuxLayout sends the current tmux layout to the client
func (wt *WebTTY) SendTmuxLayout() error {
	if wt.tmuxCtrl == nil {
		return nil
	}

	layout := wt.tmuxCtrl.GetLayout()
	if layout == nil {
		return nil
	}

	data, err := json.Marshal(layout)
	if err != nil {
		return errors.Wrap(err, "failed to marshal tmux layout")
	}

	return wt.masterWrite(append([]byte{TmuxLayoutUpdate}, data...))
}

// SendTmuxModeUpdate sends the copy mode state to the client
func (wt *WebTTY) SendTmuxModeUpdate(inCopyMode bool) error {
	state := tmux.ModeState{
		InCopyMode: inCopyMode,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "failed to marshal tmux mode state")
	}

	return wt.masterWrite(append([]byte{TmuxModeUpdate}, data...))
}

// handleTmuxMessage handles tmux-specific messages from the client
func (wt *WebTTY) handleTmuxMessage(msgType byte, payload []byte) error {
	if wt.tmuxCtrl == nil {
		return nil // Silently ignore if no tmux controller
	}

	switch msgType {
	case TmuxSelectPane:
		paneID := string(payload)
		if err := wt.tmuxCtrl.SelectPane(paneID); err != nil {
			return errors.Wrap(err, "failed to select pane")
		}
		return wt.SendTmuxLayout()

	case TmuxSelectWindow:
		windowID := string(payload)
		if err := wt.tmuxCtrl.SelectWindow(windowID); err != nil {
			return errors.Wrap(err, "failed to select window")
		}
		return wt.SendTmuxLayout()

	case TmuxSplitPane:
		horizontal := string(payload) == "h"
		if err := wt.tmuxCtrl.SplitPane(horizontal); err != nil {
			return errors.Wrap(err, "failed to split pane")
		}
		return wt.SendTmuxLayout()

	case TmuxClosePane:
		paneID := string(payload)
		if err := wt.tmuxCtrl.ClosePane(paneID); err != nil {
			return errors.Wrap(err, "failed to close pane")
		}
		return wt.SendTmuxLayout()

	case TmuxCopyMode:
		enter := string(payload) == "1"
		var err error
		if enter {
			err = wt.tmuxCtrl.EnterCopyMode()
		} else {
			err = wt.tmuxCtrl.ExitCopyMode()
		}
		if err != nil {
			return errors.Wrap(err, "failed to toggle copy mode")
		}
		return wt.SendTmuxModeUpdate(enter)

	case TmuxScrollUp:
		lines, _ := strconv.Atoi(string(payload))
		if lines <= 0 {
			lines = 1
		}
		return wt.tmuxCtrl.ScrollUp(lines)

	case TmuxScrollDown:
		lines, _ := strconv.Atoi(string(payload))
		if lines <= 0 {
			lines = 1
		}
		return wt.tmuxCtrl.ScrollDown(lines)

	case TmuxNewWindow:
		if err := wt.tmuxCtrl.NewWindow(); err != nil {
			return errors.Wrap(err, "failed to create new window")
		}
		return wt.SendTmuxLayout()

	case TmuxSwitchSession:
		sessionName := string(payload)
		if err := wt.tmuxCtrl.SwitchSession(sessionName); err != nil {
			return errors.Wrap(err, "failed to switch session")
		}
		return wt.SendTmuxLayout()

	default:
		return errors.Errorf("unknown tmux message type: %c", msgType)
	}
}

// isTmuxMessage returns true if the message type is a tmux-specific message
func isTmuxMessage(msgType byte) bool {
	switch msgType {
	case TmuxSelectPane, TmuxSelectWindow, TmuxSplitPane, TmuxClosePane,
		TmuxCopyMode, TmuxSendCommand, TmuxScrollUp, TmuxScrollDown, TmuxNewWindow,
		TmuxSwitchSession:
		return true
	default:
		return false
	}
}
