package validation

import "testing"

func TestPollCleansAndAcceptsValidInput(t *testing.T) {
	question, options, err := Poll("  Which   route should we take? ", []string{" Fast path ", "Scenic path"})
	if err != nil {
		t.Fatalf("expected valid poll, got %v", err)
	}
	if question != "Which route should we take?" {
		t.Fatalf("unexpected question: %q", question)
	}
	if options[0] != "Fast path" || options[1] != "Scenic path" {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestPollRejectsDuplicateOptions(t *testing.T) {
	_, _, err := Poll("Where should we meet?", []string{"Cafe", " cafe "})
	if err == nil {
		t.Fatal("expected duplicate options to be rejected")
	}
}

func TestPasswordPolicy(t *testing.T) {
	if err := Password("alllettersbutlong"); err == nil {
		t.Fatal("expected a password without a number to be rejected")
	}
	if err := Password("decide2026now"); err != nil {
		t.Fatalf("expected valid password, got %v", err)
	}
}
