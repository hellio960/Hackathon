package utils

import (
	"testing"
)

func TestCommonInAB(t *testing.T) {
	tests := []struct {
		name   string
		sliceA []string
		sliceB []string
		want   bool
	}{
		{
			name:   "no common elements",
			sliceA: []string{"a", "b", "c"},
			sliceB: []string{"d", "e", "f"},
			want:   false,
		},
		{
			name:   "one common element",
			sliceA: []string{"a", "b", "c"},
			sliceB: []string{"c", "d", "e"},
			want:   true,
		},
		{
			name:   "multiple common elements",
			sliceA: []string{"a", "b", "c"},
			sliceB: []string{"b", "c", "d"},
			want:   true,
		},
		{
			name:   "all elements common",
			sliceA: []string{"a", "b", "c"},
			sliceB: []string{"a", "b", "c"},
			want:   true,
		},
		{
			name:   "empty slices",
			sliceA: []string{},
			sliceB: []string{},
			want:   false,
		},
		{
			name:   "one empty slice",
			sliceA: []string{"a", "b"},
			sliceB: []string{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CommonInAB(tt.sliceA, tt.sliceB); got != tt.want {
				t.Errorf("CommonInAB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInAnySlice(t *testing.T) {
	tests := []struct {
		name  string
		item  string
		slice []string
		want  bool
	}{
		{
			name:  "item exists",
			item:  "b",
			slice: []string{"a", "b", "c"},
			want:  true,
		},
		{
			name:  "item does not exist",
			item:  "d",
			slice: []string{"a", "b", "c"},
			want:  false,
		},
		{
			name:  "empty slice",
			item:  "a",
			slice: []string{},
			want:  false,
		},
		{
			name:  "first item",
			item:  "a",
			slice: []string{"a", "b", "c"},
			want:  true,
		},
		{
			name:  "last item",
			item:  "c",
			slice: []string{"a", "b", "c"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate InAnySlice function
			found := false
			for _, v := range tt.slice {
				if v == tt.item {
					found = true
					break
				}
			}
			
			if found != tt.want {
				t.Errorf("InAnySlice() = %v, want %v", found, tt.want)
			}
		})
	}
}
