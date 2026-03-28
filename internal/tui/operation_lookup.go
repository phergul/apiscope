package tui

import (
	"slices"

	"github.com/phergul/apiscope/internal/model"
)

func (m *Model) operationByKey(key model.OperationKey) *model.Operation {
	if m.session.Spec == nil {
		return nil
	}

	for index := range m.session.Spec.Operations {
		operation := &m.session.Spec.Operations[index]
		if operation.Key == key {
			return operation
		}
	}

	return nil
}

func (m *Model) resolvedSelectedOperation() *model.Operation {
	if operation := m.operationByKey(m.session.SelectedOperationKey); operation != nil {
		if len(m.viewState.VisibleOperationKeys) == 0 || slices.Contains(m.viewState.VisibleOperationKeys, operation.Key) {
			return operation
		}
	}
	if len(m.viewState.VisibleOperationKeys) == 0 {
		return nil
	}

	return m.operationByKey(m.viewState.VisibleOperationKeys[0])
}

func (m *Model) effectiveSecurityRequirement(operation *model.Operation) *model.SecurityRequirement {
	if operation != nil && operation.Security != nil {
		return operation.Security
	}

	if m.session.Spec == nil {
		return nil
	}

	return m.session.Spec.Security
}
