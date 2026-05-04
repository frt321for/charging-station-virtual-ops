package maintenance

import "testing"

func TestValidTransition(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		want    bool
	}{
		{name: "assign open", current: "open", target: "assigned", want: true},
		{name: "accept assigned", current: "assigned", target: "accepted", want: true},
		{name: "return retest to handling", current: "retest", target: "handling", want: true},
		{name: "close recovered", current: "recovered", target: "closed", want: true},
		{name: "reject skip", current: "open", target: "closed", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validTransition(test.current, test.target); got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestValidStatus(t *testing.T) {
	if !validStatus("handling") {
		t.Fatalf("expected handling to be valid")
	}
	if validStatus("done") {
		t.Fatalf("expected done to be invalid")
	}
}
