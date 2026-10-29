package remediation

import "testing"

type fakeImpl struct{}

func (fakeImpl) Run(ctx Context) (Result, error) { return Result{}, nil }

func TestRegisterAndLookup(t *testing.T) {
	kind := "test.fake_kind_register_and_lookup"
	if _, ok := Lookup(kind); ok {
		t.Fatalf("kind %q should not be registered yet", kind)
	}

	impl := fakeImpl{}
	Register(kind, impl)

	got, ok := Lookup(kind)
	if !ok {
		t.Fatalf("expected %q to be registered", kind)
	}
	if got != Implementation(impl) {
		t.Errorf("Lookup returned a different implementation than registered")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	kind := "test.fake_kind_duplicate"
	Register(kind, fakeImpl{})

	defer func() {
		if recover() == nil {
			t.Error("expected Register to panic on a duplicate kind")
		}
	}()
	Register(kind, fakeImpl{})
}
