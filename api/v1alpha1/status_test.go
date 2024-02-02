package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const mockErrorMessage = "mock error message"

func Test_DetermineEventingAuthState(t *testing.T) {
	tests := []struct {
		name        string
		givenStatus EventingAuthStatus
		wantState   State
	}{
		{
			name: "Should be ready if both conditions are true",
			givenStatus: EventingAuthStatus{
				Conditions: createTwoTrueConditions(),
			},
			wantState: StateReady,
		},
		{
			name: "Should not be ready if both conditions are false",
			givenStatus: EventingAuthStatus{
				Conditions: createTwoFalseConditions(),
			},
			wantState: StateNotReady,
		},
		{
			name: "Should not be ready if one of the conditions is false",
			givenStatus: EventingAuthStatus{
				Conditions: createTwoConditionsWithOneFalse(),
			},
			wantState: StateNotReady,
		},
		{
			name: "Should not be ready if only application condition is available",
			givenStatus: EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:   string(ConditionApplicationReady),
						Status: kmetav1.ConditionTrue,
					},
				},
			},
			wantState: StateNotReady,
		},
		{
			name: "Should not be ready if only secret condition is available",
			givenStatus: EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:   string(ConditionSecretReady),
						Status: kmetav1.ConditionTrue,
					},
				},
			},
			wantState: StateNotReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualState := determineEventingAuthState(tt.givenStatus)
			// then
			require.Equal(t, tt.wantState, actualState)
		})
	}
}

func Test_IsEventingAuthStatusEqual(t *testing.T) {
	tests := []struct {
		name           string
		givenOldStatus EventingAuthStatus
		givenNewStatus EventingAuthStatus
		result         bool
	}{
		{
			name:           "Should status be equal",
			givenOldStatus: createEventingAuthStatus(kmetav1.ConditionTrue, "mock-app1", "mock-secret-ns1", StateReady),
			givenNewStatus: createEventingAuthStatus(kmetav1.ConditionTrue, "mock-app1", "mock-secret-ns1", StateReady),
			result:         true,
		},
		{
			name:           "Should not be equal as conditions are different",
			givenOldStatus: createEventingAuthStatus(kmetav1.ConditionFalse, "mock-app1", "mock-secret-ns1", StateReady),
			givenNewStatus: createEventingAuthStatus(kmetav1.ConditionTrue, "mock-app1", "mock-secret-ns1", StateReady),
			result:         false,
		},
		{
			name:           "Should not be equal as ias app and secret names are different",
			givenOldStatus: createEventingAuthStatus(kmetav1.ConditionTrue, "mock-app1", "mock-secret-ns1", StateReady),
			givenNewStatus: createEventingAuthStatus(kmetav1.ConditionTrue, "mock-app2", "mock-secret-ns2", StateReady),
			result:         false,
		},
		{
			name:           "Should not be equal as state is different",
			givenOldStatus: createEventingAuthStatus(kmetav1.ConditionTrue, "mock-app1", "mock-secret-ns1", StateReady),
			givenNewStatus: createEventingAuthStatus(kmetav1.ConditionTrue, "mock-app1", "mock-secret-ns1", StateNotReady),
			result:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResult := IsEventingAuthStatusEqual(tt.givenOldStatus, tt.givenNewStatus)
			// then
			require.Equal(t, tt.result, actualResult)
		})
	}
}

func Test_MakeApplicationReadyCondition(t *testing.T) {
	tests := []struct {
		name              string
		givenEventingAuth *EventingAuth
		givenErr          error
		wantConditions    []kmetav1.Condition
	}{
		{
			name:              "Should application ready condition be added",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{Conditions: []kmetav1.Condition{}}),
			wantConditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
			},
		},
		{
			name: "Should update false condition to true if no error",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{Conditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionFalse,
					Reason:  ConditionReasonApplicationCreationFailed,
					Message: mockErrorMessage,
				},
			}}),
			wantConditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
			},
		},
		{
			name: "Should update condition to false if error occurs",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{Conditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
			}}),
			givenErr: fmt.Errorf(mockErrorMessage),
			wantConditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionFalse,
					Reason:  ConditionReasonApplicationCreationFailed,
					Message: mockErrorMessage,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualConditions := MakeApplicationReadyCondition(tt.givenEventingAuth, tt.givenErr)
			// then
			require.True(t, ConditionsEqual(tt.wantConditions, actualConditions))
		})
	}
}

func Test_MakeSecretReadyCondition(t *testing.T) {
	tests := []struct {
		name              string
		givenEventingAuth *EventingAuth
		givenErr          error
		wantConditions    []kmetav1.Condition
	}{
		{
			name: "Should application ready condition be added",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{Conditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
			}}),
			wantConditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
				{
					Type:    string(ConditionSecretReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonSecretCreated,
					Message: ConditionMessageSecretCreated,
				},
			},
		},
		{
			name: "Should update not ready secret condition to ready condition",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{Conditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
				{
					Type:    string(ConditionSecretReady),
					Status:  kmetav1.ConditionFalse,
					Reason:  ConditionReasonSecretCreationFailed,
					Message: mockErrorMessage,
				},
			}}),
			wantConditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
				{
					Type:    string(ConditionSecretReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonSecretCreated,
					Message: ConditionMessageSecretCreated,
				},
			},
		},
		{
			name: "Should update ready secret condition to not ready when error occurs",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{Conditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
				{
					Type:    string(ConditionSecretReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonSecretCreated,
					Message: ConditionMessageSecretCreated,
				},
			}}),
			givenErr: fmt.Errorf(mockErrorMessage),
			wantConditions: []kmetav1.Condition{
				{
					Type:    string(ConditionApplicationReady),
					Status:  kmetav1.ConditionTrue,
					Reason:  ConditionReasonApplicationCreated,
					Message: ConditionMessageApplicationCreated,
				},
				{
					Type:    string(ConditionSecretReady),
					Status:  kmetav1.ConditionFalse,
					Reason:  ConditionReasonSecretCreationFailed,
					Message: mockErrorMessage,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualConditions := MakeSecretReadyCondition(tt.givenEventingAuth, tt.givenErr)
			// then
			require.True(t, ConditionsEqual(tt.wantConditions, actualConditions))
		})
	}
}

func Test_UpdateConditionAndState(t *testing.T) {
	const invalidConditionType = "InvalidConditionType"
	tests := []struct {
		name              string
		givenEventingAuth *EventingAuth
		conditionType     ConditionType
		givenErr          error
		wantStatus        EventingAuthStatus
		wantError         error
	}{
		{
			name: "Should update state to NotReady as error occurs",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:    string(ConditionApplicationReady),
						Status:  kmetav1.ConditionTrue,
						Reason:  ConditionReasonApplicationCreated,
						Message: ConditionMessageApplicationCreated,
					},
					{
						Type:    string(ConditionSecretReady),
						Status:  kmetav1.ConditionTrue,
						Reason:  ConditionReasonSecretCreated,
						Message: ConditionMessageSecretCreated,
					},
				},
				State: StateReady,
			}),
			conditionType: ConditionSecretReady,
			givenErr:      fmt.Errorf(mockErrorMessage),
			wantStatus: EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:    string(ConditionApplicationReady),
						Status:  kmetav1.ConditionTrue,
						Reason:  ConditionReasonApplicationCreated,
						Message: ConditionMessageApplicationCreated,
					},
					{
						Type:    string(ConditionSecretReady),
						Status:  kmetav1.ConditionFalse,
						Reason:  ConditionReasonSecretCreationFailed,
						Message: mockErrorMessage,
					},
				},
				State: StateNotReady,
			},
		},
		{
			name: "Should update state to Ready as no error occurs",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:    string(ConditionApplicationReady),
						Status:  kmetav1.ConditionTrue,
						Reason:  ConditionReasonApplicationCreated,
						Message: ConditionMessageApplicationCreated,
					},
					{
						Type:    string(ConditionSecretReady),
						Status:  kmetav1.ConditionFalse,
						Reason:  ConditionReasonSecretCreationFailed,
						Message: mockErrorMessage,
					},
				},
				State: StateNotReady,
			}),
			conditionType: ConditionSecretReady,
			wantStatus: EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:    string(ConditionApplicationReady),
						Status:  kmetav1.ConditionTrue,
						Reason:  ConditionReasonApplicationCreated,
						Message: ConditionMessageApplicationCreated,
					},
					{
						Type:    string(ConditionSecretReady),
						Status:  kmetav1.ConditionTrue,
						Reason:  ConditionReasonSecretCreated,
						Message: ConditionMessageSecretCreated,
					},
				},
				State: StateReady,
			},
		},
		{
			name: "Should update state to NotReady due to missing secret condition",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:    string(ConditionApplicationReady),
						Status:  kmetav1.ConditionFalse,
						Reason:  ConditionReasonApplicationCreated,
						Message: "IAS application creation failed.",
					},
				},
				State: StateReady,
			}),
			conditionType: ConditionApplicationReady,
			wantStatus: EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:    string(ConditionApplicationReady),
						Status:  kmetav1.ConditionTrue,
						Reason:  ConditionReasonApplicationCreated,
						Message: ConditionMessageApplicationCreated,
					},
				},
				State: StateNotReady,
			},
		},
		{
			name: "Should fail if invalid condition type is provided",
			givenEventingAuth: createEventingAuthWith(EventingAuthStatus{
				Conditions: []kmetav1.Condition{
					{
						Type:    string(ConditionApplicationReady),
						Status:  kmetav1.ConditionFalse,
						Reason:  ConditionReasonApplicationCreated,
						Message: "IAS application creation failed.",
					},
				},
				State: StateNotReady,
			}),
			conditionType: invalidConditionType,
			wantError:     fmt.Errorf("unsupported condition type: %s", invalidConditionType),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualStatus, err := UpdateConditionAndState(tt.givenEventingAuth, tt.conditionType, tt.givenErr)
			// then
			if tt.wantError != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantError, err.Error())
			} else {
				require.True(t, ConditionsEqual(tt.wantStatus.Conditions, actualStatus.Conditions))
				require.True(t, ConditionsEqual(tt.givenEventingAuth.Status.Conditions, actualStatus.Conditions))
				require.Equal(t, tt.wantStatus.State, actualStatus.State)
			}
		})
	}
}

func createTwoTrueConditions() []kmetav1.Condition {
	return []kmetav1.Condition{
		{
			Type:   string(ConditionApplicationReady),
			Status: kmetav1.ConditionTrue,
		},
		{
			Type:   string(ConditionSecretReady),
			Status: kmetav1.ConditionTrue,
		},
	}
}
func createTwoFalseConditions() []kmetav1.Condition {
	return []kmetav1.Condition{
		{
			Type:   string(ConditionApplicationReady),
			Status: kmetav1.ConditionFalse,
		},
		{
			Type:   string(ConditionSecretReady),
			Status: kmetav1.ConditionFalse,
		},
	}
}

func createTwoConditionsWithOneFalse() []kmetav1.Condition {
	return []kmetav1.Condition{
		{
			Type:   string(ConditionApplicationReady),
			Status: kmetav1.ConditionTrue,
		},
		{
			Type:   string(ConditionSecretReady),
			Status: kmetav1.ConditionFalse,
		},
	}
}

func createEventingAuthStatus(secretReadyStatus kmetav1.ConditionStatus, appName, secretNSName string, state State) EventingAuthStatus {
	return EventingAuthStatus{
		Conditions: []kmetav1.Condition{
			{
				Type:   string(ConditionApplicationReady),
				Status: kmetav1.ConditionTrue,
			},
			{
				Type:   string(ConditionSecretReady),
				Status: secretReadyStatus,
			},
		},
		Application: &IASApplication{
			Name: appName,
			UUID: "mock-uuid",
		},
		AuthSecret: &AuthSecret{
			NamespacedName: secretNSName,
			ClusterId:      "mock-cluster-reference",
		},
		State: state,
	}
}

func createEventingAuthWith(status EventingAuthStatus) *EventingAuth {
	return &EventingAuth{
		ObjectMeta: kmetav1.ObjectMeta{
			Namespace: "mock-eauth-ns",
			Name:      "mock-eauth-name",
		},
		Status: status,
	}
}
