package tmux

// Session represents a tmux session
type Session struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Windows  int    `json:"windows"`
	Attached bool   `json:"attached"`
	Active   bool   `json:"active"`
}

// Layout represents the complete tmux state
type Layout struct {
	SessionID    string    `json:"sessionId"`
	SessionName  string    `json:"sessionName"`
	Sessions     []Session `json:"sessions"`
	Windows      []Window  `json:"windows"`
	ActiveWinID  string    `json:"activeWindowId"`
	ActivePaneID string    `json:"activePaneId"`
}

// Window represents a tmux window
type Window struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Index  int    `json:"index"`
	Active bool   `json:"active"`
	Panes  []Pane `json:"panes"`
}

// Pane represents a tmux pane
type Pane struct {
	ID      string `json:"id"`
	Index   int    `json:"index"`
	Active  bool   `json:"active"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Top     int    `json:"top"`
	Left    int    `json:"left"`
	Command string `json:"command"`
	Title   string `json:"title"`
}

// ModeState represents the current mode of a pane (normal, copy, etc.)
type ModeState struct {
	PaneID         string `json:"paneId"`
	InCopyMode     bool   `json:"inCopyMode"`
	ScrollPosition int    `json:"scrollPosition"`
	HistorySize    int    `json:"historySize"`
}

// Event represents a tmux control mode event
type Event struct {
	Type    string
	Payload string
}
