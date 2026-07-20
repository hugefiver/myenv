package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/profiles"
)

type profileRequestKind uint8

const (
	profileRequestImportFile profileRequestKind = iota + 1
	profileRequestImportURL
	profileRequestActivation
	profileRequestRefresh
	profileRequestDelete
)

const (
	profileDownloadTimeout   = 90 * time.Second
	profileActivationTimeout = 30 * time.Second
)

type profileRequest struct {
	ID     requestID
	Target string
	Kind   profileRequestKind
}

type profileResultMsg struct {
	request        profileRequest
	entry          profiles.Entry
	persisted      bool
	runtimeApplied bool
	err            error
}

func (m *profilesModel) beginProfileRequest(kind profileRequestKind, target string) (profileRequest, bool) {
	if m.busy {
		return profileRequest{}, false
	}
	m.requestNo++
	m.request = profileRequest{ID: m.requestNo, Target: target, Kind: kind}
	m.busy = true
	return m.request, true
}

func (m *profilesModel) importEntry(input string) (tea.Cmd, string, bool) {
	kind := profileRequestImportFile
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		kind = profileRequestImportURL
	}
	request, ok := m.beginProfileRequest(kind, input)
	if !ok {
		return nil, "profile operation already in progress", true
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, profileDownloadTimeout)
		defer cancel()
		var entry profiles.Entry
		var err error
		if kind == profileRequestImportURL {
			entry, err = m.store.ImportURL(ctx, input)
		} else {
			entry, err = m.store.ImportFile(ctx, input)
		}
		return profileResultMsg{request: request, entry: entry, persisted: err == nil, err: err}
	}, "importing", false
}

func (m *profilesModel) applyProfile(index int) (tea.Cmd, string, bool) {
	entry, ok := m.entryAt(index)
	if !ok {
		return nil, "", false
	}
	request, ok := m.beginProfileRequest(profileRequestActivation, entry.ID)
	if !ok {
		return nil, "profile operation already in progress", true
	}
	return func() tea.Msg {
		runtimeApplied, err := m.activate(entry.ID)
		return profileResultMsg{
			request:        request,
			entry:          entry,
			persisted:      err == nil,
			runtimeApplied: runtimeApplied,
			err:            err,
		}
	}, "activating profile", false
}

func (m *profilesModel) refreshURL(index int) (tea.Cmd, string, bool) {
	entry, ok := m.entryAt(index)
	if !ok {
		return nil, "", false
	}
	if entry.Kind != profiles.KindURL {
		return nil, "profile is not URL-backed", true
	}
	request, ok := m.beginProfileRequest(profileRequestRefresh, entry.ID)
	if !ok {
		return nil, "profile operation already in progress", true
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, profileDownloadTimeout)
		refreshed, err := m.store.RefreshURL(ctx, entry.ID)
		cancel()
		if err != nil {
			return profileResultMsg{request: request, entry: entry, err: err}
		}
		runtimeApplied, activationErr := m.activate(entry.ID)
		return profileResultMsg{
			request:        request,
			entry:          refreshed,
			persisted:      true,
			runtimeApplied: runtimeApplied,
			err:            activationErr,
		}
	}, "refreshing URL", false
}

func (m *profilesModel) deleteProfile(index int) (tea.Cmd, string, bool) {
	entry, ok := m.entryAt(index)
	if !ok {
		return nil, "", false
	}
	request, ok := m.beginProfileRequest(profileRequestDelete, entry.ID)
	if !ok {
		return nil, "profile operation already in progress", true
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, reqTimeout)
		defer cancel()
		if err := ctx.Err(); err != nil {
			return profileResultMsg{request: request, entry: entry, err: err}
		}
		err := m.store.Delete(ctx, entry.ID)
		return profileResultMsg{request: request, entry: entry, persisted: err == nil, err: err}
	}, "deleting profile", false
}

func (m *profilesModel) activate(id string) (bool, error) {
	ctx, cancel := context.WithTimeout(m.ctx, profileActivationTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	txn, err := m.store.PrepareActivation(id)
	if err != nil {
		return false, err
	}
	if err := m.cli.PutConfig(ctx, txn.Payload); err != nil {
		if abortErr := m.store.AbortActivation(txn); abortErr != nil {
			return false, fmt.Errorf("%w; abort activation: %v", err, abortErr)
		}
		return false, err
	}
	if err := m.store.CommitActivation(txn); err != nil {
		return true, err
	}
	return true, nil
}

func (m *profilesModel) handleProfileResult(result profileResultMsg) (tea.Cmd, string, bool) {
	if result.request != m.request {
		if result.persisted {
			cursor := m.cursor
			m.syncFromStore()
			m.cursor = normalizedCursor(cursor, len(m.entries))
		}
		return nil, "", false
	}
	m.busy = false
	m.request = profileRequest{}
	m.syncFromStore()

	name := result.entry.Name
	switch result.request.Kind {
	case profileRequestImportFile, profileRequestImportURL:
		if result.err != nil {
			return nil, fmt.Sprintf("import failed: %v", result.err), true
		}
		return nil, fmt.Sprintf("imported: %s", name), false
	case profileRequestActivation:
		return activationStatus("activation failed", name, result)
	case profileRequestRefresh:
		if result.err == nil {
			return nil, fmt.Sprintf("refreshed and activated: %s", name), false
		}
		if result.runtimeApplied {
			return partialActivationStatus(result.err)
		}
		if result.persisted {
			return nil, fmt.Sprintf("refreshed content stored; activation failed: %v", result.err), true
		}
		return nil, fmt.Sprintf("refresh failed: %v", result.err), true
	case profileRequestDelete:
		if errors.Is(result.err, profiles.ErrActiveProfile) {
			return nil, "cannot delete active profile", true
		}
		if result.err != nil {
			return nil, fmt.Sprintf("delete failed: %v", result.err), true
		}
		return nil, fmt.Sprintf("deleted: %s", name), false
	}
	return nil, "", false
}

func activationStatus(failure, name string, result profileResultMsg) (tea.Cmd, string, bool) {
	if result.err == nil {
		return nil, fmt.Sprintf("activated: %s", name), false
	}
	if result.runtimeApplied {
		return partialActivationStatus(result.err)
	}
	return nil, fmt.Sprintf("%s: %s: %v", failure, name, result.err), true
}

func partialActivationStatus(err error) (tea.Cmd, string, bool) {
	if errors.Is(err, profiles.ErrActiveIDPersistence) {
		return nil, fmt.Sprintf("runtime and boot snapshot updated; active metadata persistence failed: %v", err), true
	}
	return nil, fmt.Sprintf("runtime activation succeeded; boot snapshot update failed: %v", err), true
}
