package v1alpha1

import (
	"fmt"
	"reflect"

	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	ConditionApplicationReady ConditionType = "IASApplicationReady"
	ConditionSecretReady      ConditionType = "SecretReady"
)

type ConditionReason string

const (
	ConditionReasonApplicationCreated        string = "IASApplicationCreated"
	ConditionReasonSecretCreated             string = "SecretCreated"
	ConditionReasonApplicationCreationFailed string = "IASApplicationCreationFailed"
	ConditionReasonSecretCreationFailed      string = "SecretCreationFailed"
)

const (
	ConditionMessageApplicationCreated string = "IAS application is successfully created."
	ConditionMessageSecretCreated      string = "Eventing webhook authentication secret is successfully created."
)

func UpdateConditionAndState(eventingAuth *EventingAuth, conditionType ConditionType, err error) (EventingAuthStatus, error) {
	switch conditionType {
	case ConditionApplicationReady:
		{
			eventingAuth.Status.Conditions = MakeApplicationReadyCondition(eventingAuth, err)
		}
	case ConditionSecretReady:
		{
			eventingAuth.Status.Conditions = MakeSecretReadyCondition(eventingAuth, err)
		}
	default:
		return eventingAuth.Status, fmt.Errorf("unsupported condition type: %s", conditionType)
	}

	if err != nil {
		eventingAuth.Status.State = StateNotReady
	} else {
		eventingAuth.Status.State = determineEventingAuthState(eventingAuth.Status)
	}
	return eventingAuth.Status, nil
}

// MakeApplicationReadyCondition updates the ConditionApplicationActive condition based on the given error value.
func MakeApplicationReadyCondition(eventingAuth *EventingAuth, err error) []kmetav1.Condition {
	applicationReadyCondition := kmetav1.Condition{
		Type:               string(ConditionApplicationReady),
		LastTransitionTime: kmetav1.Now(),
	}
	if err == nil {
		applicationReadyCondition.Status = kmetav1.ConditionTrue
		applicationReadyCondition.Reason = ConditionReasonApplicationCreated
		applicationReadyCondition.Message = ConditionMessageApplicationCreated
	} else {
		applicationReadyCondition.Message = err.Error()
		applicationReadyCondition.Reason = ConditionReasonApplicationCreationFailed
		applicationReadyCondition.Status = kmetav1.ConditionFalse
	}
	for ix, activeCond := range eventingAuth.Status.Conditions {
		if activeCond.Type == string(ConditionApplicationReady) {
			if applicationReadyCondition.Status == activeCond.Status &&
				applicationReadyCondition.Reason == activeCond.Reason &&
				applicationReadyCondition.Message == activeCond.Message {
				return eventingAuth.Status.Conditions
			} else {
				eventingAuth.Status.Conditions[ix] = applicationReadyCondition
				return eventingAuth.Status.Conditions
			}
		}
	}
	return append(eventingAuth.Status.Conditions, applicationReadyCondition)
}

// MakeSecretReadyCondition updates the ConditionSecretReady condition based on the given error value.
func MakeSecretReadyCondition(eventingAuth *EventingAuth, err error) []kmetav1.Condition {
	secretReadyCondition := kmetav1.Condition{
		Type:               string(ConditionSecretReady),
		LastTransitionTime: kmetav1.Now(),
	}
	if err == nil {
		secretReadyCondition.Status = kmetav1.ConditionTrue
		secretReadyCondition.Reason = ConditionReasonSecretCreated
		secretReadyCondition.Message = ConditionMessageSecretCreated
	} else {
		secretReadyCondition.Message = err.Error()
		secretReadyCondition.Reason = ConditionReasonSecretCreationFailed
		secretReadyCondition.Status = kmetav1.ConditionFalse
	}
	for ix, activeCond := range eventingAuth.Status.Conditions {
		if activeCond.Type == string(ConditionSecretReady) {
			if secretReadyCondition.Status == activeCond.Status &&
				secretReadyCondition.Reason == activeCond.Reason &&
				secretReadyCondition.Message == activeCond.Message {
				return eventingAuth.Status.Conditions
			} else {
				eventingAuth.Status.Conditions[ix] = secretReadyCondition
				return eventingAuth.Status.Conditions
			}
		}
	}
	return append(eventingAuth.Status.Conditions, secretReadyCondition)
}

// ConditionsEqual checks if two list of conditions are equal.
func ConditionsEqual(existing, expected []kmetav1.Condition) bool {
	// not equal if length is different
	if len(existing) != len(expected) {
		return false
	}

	// compile map of Conditions per ConditionType
	existingMap := make(map[string]kmetav1.Condition, len(existing))
	for _, value := range existing {
		existingMap[value.Type] = value
	}

	for _, value := range expected {
		if !ConditionEquals(existingMap[value.Type], value) {
			return false
		}
	}

	return true
}

// ConditionEquals checks if two conditions are equal.
func ConditionEquals(existing, expected kmetav1.Condition) bool {
	isTypeEqual := existing.Type == expected.Type
	isStatusEqual := existing.Status == expected.Status
	isReasonEqual := existing.Reason == expected.Reason
	isMessageEqual := existing.Message == expected.Message

	return isStatusEqual && isReasonEqual && isMessageEqual && isTypeEqual
}

func IsEventingAuthStatusEqual(oldStatus, newStatus EventingAuthStatus) bool {
	oldStatusWithoutCond := oldStatus.DeepCopy()
	newStatusWithoutCond := newStatus.DeepCopy()

	// remove conditions, so that we don't compare them
	oldStatusWithoutCond.Conditions = []kmetav1.Condition{}
	newStatusWithoutCond.Conditions = []kmetav1.Condition{}

	return reflect.DeepEqual(oldStatusWithoutCond, newStatusWithoutCond) &&
		ConditionsEqual(oldStatus.Conditions, newStatus.Conditions)
}

// determineEventingAuthState returns 'Ready' if both IAS app and secret are created, otherwise 'NoReady'.
func determineEventingAuthState(status EventingAuthStatus) State {
	var applicationReady, secretReady bool
	for _, cond := range status.Conditions {
		if cond.Type == string(ConditionApplicationReady) {
			applicationReady = cond.Status == kmetav1.ConditionTrue
		}
		if cond.Type == string(ConditionSecretReady) {
			secretReady = cond.Status == kmetav1.ConditionTrue
		}
	}
	if applicationReady && secretReady {
		return StateReady
	}
	return StateNotReady
}
