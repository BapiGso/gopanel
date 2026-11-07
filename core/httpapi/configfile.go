package httpapi

import "gopanel/core/storage"

type ConfigDocument = storage.Document

type ConfigWrite struct {
	Content  *string `json:"content" validate:"required"`
	Revision string  `json:"revision" validate:"required,len=64,hexadecimal"`
}

type AutostartWrite struct {
	Autostart *bool  `json:"autostart" validate:"required"`
	Revision  string `json:"revision" validate:"required,len=64,hexadecimal"`
}

type ServiceStatus struct {
	State            string `json:"state"`
	Autostart        bool   `json:"autostart"`
	SettingsRevision string `json:"settings_revision"`
	AppliedRevision  string `json:"applied_revision,omitempty"`
	PendingChanges   bool   `json:"pending_changes"`
	LastError        string `json:"last_error,omitempty"`
}
