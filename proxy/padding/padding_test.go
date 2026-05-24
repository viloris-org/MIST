package padding

import "testing"

func TestNewPaddingFactoryRejectsOversizedRecord(t *testing.T) {
	if NewPaddingFactory([]byte("stop=1\n0=1-16385")) != nil {
		t.Fatal("expected oversized record payload to be rejected")
	}
}

func TestNewPaddingFactoryCopiesRawScheme(t *testing.T) {
	raw := []byte("stop=1\n0=10-10")
	factory := NewPaddingFactory(raw)
	if factory == nil {
		t.Fatal("expected padding factory")
	}
	raw[0] = 'x'
	if string(factory.RawScheme) != "stop=1\n0=10-10" {
		t.Fatalf("raw scheme was mutated: %q", factory.RawScheme)
	}
}

func TestGenerateRecordPayloadSizesIntoReusesDestination(t *testing.T) {
	factory := NewPaddingFactory([]byte("stop=2\n1=10-10,c,20-20"))
	if factory == nil {
		t.Fatal("expected padding factory")
	}
	dst := make([]int, 0, 3)
	got := factory.GenerateRecordPayloadSizesInto(1, dst)
	if len(got) != 3 || got[0] != 10 || got[1] != CheckMark || got[2] != 20 {
		t.Fatalf("unexpected payload sizes: %#v", got)
	}
	if cap(got) != cap(dst) {
		t.Fatalf("destination was not reused: got cap %d, want %d", cap(got), cap(dst))
	}
}
