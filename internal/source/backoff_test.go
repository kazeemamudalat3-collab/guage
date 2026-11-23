package source

import (
	"testing"
	"time"
)

// TestBackoffForStaysUnderTheCeiling pins the whole doubling walk, not just its
// first two steps: the existing retry test stops at attempt 2, so an off-by-one
// in the loop bound, a missing cap for a large base, or a base that never
// reaches the ceiling would all pass CI. Durations are asserted, never slept.
func TestBackoffForStaysUnderTheCeiling(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		base time.Duration
		want []time.Duration
	}{
		{
			name: "one millisecond",
			base: time.Millisecond,
			want: []time.Duration{
				time.Millisecond,
				2 * time.Millisecond,
				4 * time.Millisecond,
				8 * time.Millisecond,
				16 * time.Millisecond,
				32 * time.Millisecond,
				64 * time.Millisecond,
				128 * time.Millisecond,
				256 * time.Millisecond,
				512 * time.Millisecond,
				1024 * time.Millisecond,
				2048 * time.Millisecond,
			},
		},
		{
			name: "two hundred milliseconds",
			base: 200 * time.Millisecond,
			want: []time.Duration{
				200 * time.Millisecond,
				400 * time.Millisecond,
				800 * time.Millisecond,
				1600 * time.Millisecond,
				3200 * time.Millisecond,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
			},
		},
		{
			// The 4 s base clamps on its second attempt: 8 s would cross the
			// ceiling, so it is reported as 5 s and never as 8 s.
			name: "four seconds",
			base: 4 * time.Second,
			want: []time.Duration{
				4 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
				5 * time.Second,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if len(tc.want) != 12 {
				t.Fatalf("the table covers %d attempts, want 12", len(tc.want))
			}
			if got := backoffFor(tc.base, 1); got != tc.base {
				t.Fatalf("backoffFor(%v, 1) = %v, want the base unchanged", tc.base, got)
			}
			var previous time.Duration
			for i, want := range tc.want {
				attempt := i + 1
				got := backoffFor(tc.base, attempt)
				if got != want {
					t.Fatalf("backoffFor(%v, %d) = %v, want %v", tc.base, attempt, got, want)
				}
				if got <= 0 {
					t.Fatalf("backoffFor(%v, %d) = %v, want a positive delay", tc.base, attempt, got)
				}
				if got > maxBackoff {
					t.Fatalf("backoffFor(%v, %d) = %v, above the %v ceiling", tc.base, attempt, got, maxBackoff)
				}
				if got < previous {
					t.Fatalf("backoffFor(%v, %d) = %v, which is below the previous %v", tc.base, attempt, got, previous)
				}
				previous = got
			}
		})
	}

	t.Run("zero base", func(t *testing.T) {
		t.Parallel()
		if got := backoffFor(0, 1); got != 0 {
			t.Fatalf("backoffFor(0, 1) = %v, want the zero base unchanged", got)
		}
		for attempt := 1; attempt <= 12; attempt++ {
			if got := backoffFor(0, attempt); got != 0 {
				t.Fatalf("backoffFor(0, %d) = %v, want zero", attempt, got)
			}
		}
	})
}
